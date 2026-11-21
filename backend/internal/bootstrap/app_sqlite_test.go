package bootstrap

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestSQLiteApplicationSupportsAPIWritesFetchJobsAndReopen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbPath := filepath.Join(t.TempDir(), "odd # name? with spaces", "my-feed.db")
	require.NoError(t, os.MkdirAll(filepath.Dir(dbPath), 0o755))

	require.NoError(t, goose.SetDialect("sqlite3"))
	migrationDB, err := sql.Open("sqlite", sqliteTestDSN(dbPath))
	require.NoError(t, err)
	require.NoError(t, goose.Up(migrationDB, "../../migrations"))
	require.NoError(t, migrationDB.Close())

	cfg := Config{
		DatabasePath: dbPath,
		HTTPAddr:     ":0",
		Fetch: FetchConfig{
			RetryBackoff:     time.Hour,
			ScheduleInterval: time.Hour,
			ExecutorPoll:     time.Hour,
		},
	}
	app, err := NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	createSource := requestJSON(t, app.Router, http.MethodPost, "/api/v1/sources/", map[string]any{
		"name": "Example RSS", "type": "rss", "rssUrl": "https://example.test/feed.xml",
	})
	require.Equal(t, http.StatusCreated, createSource.Code)
	var source struct {
		ID string `json:"id"`
	}
	decodeResponseData(t, createSource.Body.Bytes(), &source)

	createEntry := requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/save-link", map[string]any{
		"title": "Saved article", "link": "https://example.test/article",
	})
	require.Equal(t, http.StatusCreated, createEntry.Code)

	const concurrentEntryWrites = 8
	var wg sync.WaitGroup
	entryResponses := make(chan *httptest.ResponseRecorder, concurrentEntryWrites)
	ready := make(chan struct{}, concurrentEntryWrites)
	startWrites := make(chan struct{})
	for i := 0; i < concurrentEntryWrites; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ready <- struct{}{}
			<-startWrites
			entryResponses <- requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/save-link", map[string]any{
				"title": "Concurrent entry", "link": fmt.Sprintf("https://example.test/concurrent-%d", index),
			})
		}(i)
	}
	for i := 0; i < concurrentEntryWrites; i++ {
		<-ready
	}
	close(startWrites)
	createJob := requestJSON(t, app.Router, http.MethodPost, "/api/v1/fetch-jobs/", map[string]any{"sourceId": source.ID})
	wg.Wait()
	close(entryResponses)
	for response := range entryResponses {
		require.Equal(t, http.StatusCreated, response.Code)
	}
	require.Equal(t, http.StatusCreated, createJob.Code)
	var job struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	decodeResponseData(t, createJob.Body.Bytes(), &job)
	require.Equal(t, "pending", job.Status)

	deactivate := requestJSON(t, app.Router, http.MethodPost, "/api/v1/sources/"+source.ID+"/deactivate", map[string]any{})
	require.Equal(t, http.StatusOK, deactivate.Code)
	getJob := requestJSON(t, app.Router, http.MethodGet, "/api/v1/fetch-jobs/"+job.ID, nil)
	require.Equal(t, http.StatusOK, getJob.Code)
	require.Contains(t, getJob.Body.String(), `"status":"failed"`)
	require.NoError(t, app.Pool.Close())

	app, err = NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	defer app.Close()
	getSource := requestJSON(t, app.Router, http.MethodGet, "/api/v1/sources/"+source.ID, nil)
	require.Equal(t, http.StatusOK, getSource.Code)
	require.Contains(t, getSource.Body.String(), "Example RSS")
	listEntries := requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/", nil)
	require.Equal(t, http.StatusOK, listEntries.Code)
	require.Contains(t, listEntries.Body.String(), "Saved article")
}

func TestSQLiteApplicationRunsFetchExecutorAlongsideAPIWrites(t *testing.T) {
	feedStarted := make(chan struct{})
	allowFeedResponse := make(chan struct{})
	var releaseFeed sync.Once
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		select {
		case <-feedStarted:
		default:
			close(feedStarted)
		}
		<-allowFeedResponse
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<rss version="2.0"><channel><item><title>Fetched article</title><link>https://example.test/fetched</link></item></channel></rss>`))
	}))
	defer feedServer.Close()
	defer releaseFeed.Do(func() { close(allowFeedResponse) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbPath := filepath.Join(t.TempDir(), "my-feed.db")
	require.NoError(t, goose.SetDialect("sqlite3"))
	migrationDB, err := sql.Open("sqlite", sqliteTestDSN(dbPath))
	require.NoError(t, err)
	require.NoError(t, goose.Up(migrationDB, "../../migrations"))
	require.NoError(t, migrationDB.Close())

	cfg := Config{
		DatabasePath: dbPath,
		HTTPAddr:     ":0",
		Fetch: FetchConfig{
			RetryBackoff:     time.Hour,
			ScheduleInterval: time.Hour,
			ExecutorPoll:     5 * time.Millisecond,
		},
	}
	app, err := NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	defer func() {
		cancel()
		app.Close()
	}()

	createSource := requestJSON(t, app.Router, http.MethodPost, "/api/v1/sources/", map[string]any{
		"name": "Local RSS", "type": "rss", "rssUrl": feedServer.URL,
	})
	require.Equal(t, http.StatusCreated, createSource.Code)
	var source struct {
		ID string `json:"id"`
	}
	decodeResponseData(t, createSource.Body.Bytes(), &source)
	createJob := requestJSON(t, app.Router, http.MethodPost, "/api/v1/fetch-jobs/", map[string]any{"sourceId": source.ID})
	require.Equal(t, http.StatusCreated, createJob.Code)
	var job struct {
		ID string `json:"id"`
	}
	decodeResponseData(t, createJob.Body.Bytes(), &job)

	select {
	case <-feedStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("fetch executor did not request the local RSS feed")
	}

	const concurrentEntryWrites = 8
	var wg sync.WaitGroup
	responses := make(chan *httptest.ResponseRecorder, concurrentEntryWrites)
	for i := 0; i < concurrentEntryWrites; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			responses <- requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/save-link", map[string]any{
				"title": "Concurrent entry", "link": fmt.Sprintf("https://example.test/concurrent-%d", index),
			})
		}(i)
	}
	wg.Wait()
	close(responses)
	for response := range responses {
		require.Equal(t, http.StatusCreated, response.Code)
	}
	releaseFeed.Do(func() { close(allowFeedResponse) })

	deadline := time.Now().Add(5 * time.Second)
	for {
		getJob := requestJSON(t, app.Router, http.MethodGet, "/api/v1/fetch-jobs/"+job.ID, nil)
		require.Equal(t, http.StatusOK, getJob.Code)
		if bytes.Contains(getJob.Body.Bytes(), []byte(`"status":"succeeded"`)) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("fetch job did not succeed: %s", getJob.Body.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	listEntries := requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/", nil)
	require.Equal(t, http.StatusOK, listEntries.Code)
	require.Contains(t, listEntries.Body.String(), "Fetched article")
}

func TestSQLiteOnlineBackupRestoresAPIState(t *testing.T) {
	sqlite3, err := exec.LookPath("sqlite3")
	if err != nil {
		t.Skip("sqlite3 CLI is required to exercise the documented .backup command")
	}
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "my-feed.db")
	backupPath := filepath.Join(t.TempDir(), "my-feed-backup.db")
	require.NoError(t, goose.SetDialect("sqlite3"))
	migrationDB, err := sql.Open("sqlite", sqliteTestDSN(dbPath))
	require.NoError(t, err)
	require.NoError(t, goose.Up(migrationDB, "../../migrations"))
	require.NoError(t, migrationDB.Close())
	cfg := Config{DatabasePath: dbPath, HTTPAddr: ":0", Fetch: FetchConfig{
		RetryBackoff: time.Hour, ScheduleInterval: time.Hour, ExecutorPoll: time.Hour,
	}}
	app, err := NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	defer func() { app.Close() }()

	createSource := requestJSON(t, app.Router, http.MethodPost, "/api/v1/sources/", map[string]any{
		"name": "Backup RSS", "type": "rss", "rssUrl": "https://example.test/backup.xml",
	})
	require.Equal(t, http.StatusCreated, createSource.Code)
	var source struct {
		ID string `json:"id"`
	}
	decodeResponseData(t, createSource.Body.Bytes(), &source)
	createEntry := requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/save-link", map[string]any{
		"title": "Backup entry", "link": "https://example.test/backup-entry",
	})
	require.Equal(t, http.StatusCreated, createEntry.Code)
	var entry struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createEntry.Body.Bytes(), &entry))
	require.Equal(t, http.StatusOK, requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/"+entry.Data.ID+"/read", map[string]any{}).Code)
	require.Equal(t, http.StatusOK, requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/"+entry.Data.ID+"/save", map[string]any{}).Code)
	createJob := requestJSON(t, app.Router, http.MethodPost, "/api/v1/fetch-jobs/", map[string]any{"sourceId": source.ID})
	require.Equal(t, http.StatusCreated, createJob.Code)
	var job struct {
		ID string `json:"id"`
	}
	decodeResponseData(t, createJob.Body.Bytes(), &job)

	// This is the same online backup operation documented in README.md, while the API owns an open DB.
	command := exec.Command(sqlite3, dbPath, ".backup '"+backupPath+"'")
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	require.FileExists(t, backupPath)
	assertSQLiteIntegrity(t, backupPath)
	require.NoError(t, app.Pool.Close())

	require.NoError(t, os.Remove(dbPath))
	for _, suffix := range []string{"-wal", "-shm"} {
		err = os.Remove(dbPath + suffix)
		require.True(t, err == nil || os.IsNotExist(err), "remove stale SQLite sidecar: %v", err)
	}
	backup, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(dbPath, backup, 0o600))
	app, err = NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	assertSQLiteIntegrity(t, dbPath)

	getSource := requestJSON(t, app.Router, http.MethodGet, "/api/v1/sources/"+source.ID, nil)
	require.Equal(t, http.StatusOK, getSource.Code)
	require.Contains(t, getSource.Body.String(), "Backup RSS")
	getEntry := requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/"+entry.Data.ID, nil)
	require.Equal(t, http.StatusOK, getEntry.Code)
	require.Contains(t, getEntry.Body.String(), `"readAt":"`)
	require.Contains(t, getEntry.Body.String(), `"savedAt":"`)
	getJob := requestJSON(t, app.Router, http.MethodGet, "/api/v1/fetch-jobs/"+job.ID, nil)
	require.Equal(t, http.StatusOK, getJob.Code)
	require.Contains(t, getJob.Body.String(), `"status":"pending"`)
}

func assertSQLiteIntegrity(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", sqliteTestDSN(path))
	require.NoError(t, err)
	defer db.Close()
	var result string
	require.NoError(t, db.QueryRow("PRAGMA integrity_check").Scan(&result))
	require.Equal(t, "ok", result)
}

func requestJSON(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var input []byte
	if body != nil {
		var err error
		input, err = json.Marshal(body)
		require.NoError(t, err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(input))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func sqliteTestDSN(path string) string {
	return (&url.URL{Scheme: "file", Path: path, RawQuery: "_pragma=foreign_keys(1)"}).String()
}

func TestEntryListCursorEnvelopeAndOrdering(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbPath := filepath.Join(t.TempDir(), "entries.db")
	require.NoError(t, goose.SetDialect("sqlite3"))
	db, err := sql.Open("sqlite", sqliteTestDSN(dbPath))
	require.NoError(t, err)
	require.NoError(t, goose.Up(db, "../../migrations"))
	require.NoError(t, db.Close())
	app, err := NewWithConfig(ctx, Config{DatabasePath: dbPath, HTTPAddr: ":0", Fetch: FetchConfig{RetryBackoff: time.Hour, ScheduleInterval: time.Hour, ExecutorPoll: time.Hour}})
	require.NoError(t, err)
	defer app.Close()
	for _, item := range []struct{ title, published string }{{"older", "2020-01-01T00:00:00Z"}, {"tie one", "2021-01-01T00:00:00Z"}, {"tie two", "2021-01-01T00:00:00Z"}, {"undated", ""}} {
		body := map[string]any{"title": item.title, "link": "https://example.test/" + url.PathEscape(item.title)}
		if item.published != "" {
			body["publishedAt"] = item.published
		}
		response := requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/save-link", body)
		require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	}
	type page struct {
		Data []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"data"`
		Meta struct {
			NextCursor string `json:"nextCursor"`
		} `json:"meta"`
	}
	firstResponse := requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/?pageSize=2", nil)
	require.Equal(t, http.StatusOK, firstResponse.Code)
	var first page
	require.NoError(t, json.Unmarshal(firstResponse.Body.Bytes(), &first))
	require.Len(t, first.Data, 2)
	require.Equal(t, "undated", first.Data[0].Title)

	require.NotEmpty(t, first.Meta.NextCursor)
	require.NotContains(t, firstResponse.Body.String(), "totalCount")
	secondResponse := requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/?pageSize=2&cursor="+url.QueryEscape(first.Meta.NextCursor), nil)
	var second page
	require.NoError(t, json.Unmarshal(secondResponse.Body.Bytes(), &second))
	require.Len(t, second.Data, 2)
	require.Greater(t, first.Data[1].ID, second.Data[0].ID)
	require.Equal(t, "older", second.Data[1].Title)

	require.Empty(t, second.Meta.NextCursor)
	require.NotContains(t, secondResponse.Body.String(), `"nextCursor"`)
}

func TestEntryReadAndSavedMutationsPersistInFilteredLists(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbPath := filepath.Join(t.TempDir(), "state.db")
	require.NoError(t, goose.SetDialect("sqlite3"))
	db, err := sql.Open("sqlite", sqliteTestDSN(dbPath))
	require.NoError(t, err)
	require.NoError(t, goose.Up(db, "../../migrations"))
	require.NoError(t, db.Close())
	cfg := Config{DatabasePath: dbPath, HTTPAddr: ":0", Fetch: FetchConfig{RetryBackoff: time.Hour, ScheduleInterval: time.Hour, ExecutorPoll: time.Hour}}
	app, err := NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	defer func() { app.Close() }()
	created := requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/save-link", map[string]any{"title": "State test", "link": "https://example.test/state"})
	require.Equal(t, http.StatusCreated, created.Code)
	var entry struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &entry))
	list := func(criteria string) []string {
		t.Helper()
		response := requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/?"+criteria, nil)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var page struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &page))
		ids := make([]string, 0, len(page.Data))
		for _, item := range page.Data {
			ids = append(ids, item.ID)
		}
		return ids
	}
	mutate := func(action string) {
		t.Helper()
		response := requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/"+entry.Data.ID+"/"+action, nil)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	}
	require.Contains(t, list("unread=true"), entry.Data.ID)
	require.NotContains(t, list("savedOnly=true"), entry.Data.ID)
	mutate("save")
	require.Contains(t, list("unread=true&savedOnly=true"), entry.Data.ID)
	mutate("read")
	require.NotContains(t, list("unread=true"), entry.Data.ID)
	require.Contains(t, list("savedOnly=true"), entry.Data.ID)
	app.Close()
	app, err = NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	require.NotContains(t, list("unread=true"), entry.Data.ID)
	require.Contains(t, list("savedOnly=true"), entry.Data.ID)
	mutate("clear-read")
	require.Contains(t, list("unread=true&savedOnly=true"), entry.Data.ID)
	mutate("clear-save")
	require.Contains(t, list("unread=true"), entry.Data.ID)
	require.NotContains(t, list("savedOnly=true"), entry.Data.ID)
}

func TestEntryListFiltersAndCursor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbPath := filepath.Join(t.TempDir(), "filters.db")
	require.NoError(t, goose.SetDialect("sqlite3"))
	db, err := sql.Open("sqlite", sqliteTestDSN(dbPath))
	require.NoError(t, err)
	require.NoError(t, goose.Up(db, "../../migrations"))
	require.NoError(t, db.Close())
	app, err := NewWithConfig(ctx, Config{DatabasePath: dbPath, HTTPAddr: ":0", Fetch: FetchConfig{RetryBackoff: time.Hour, ScheduleInterval: time.Hour, ExecutorPoll: time.Hour}})
	require.NoError(t, err)
	defer app.Close()
	var ids []string
	for _, title := range []string{"unread", "read"} {
		response := requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/save-link", map[string]any{"title": title, "link": "https://example.test/" + title})
		require.Equal(t, http.StatusCreated, response.Code)
		var entry struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &entry))
		ids = append(ids, entry.Data.ID)
	}
	require.Equal(t, http.StatusOK, requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/"+ids[1]+"/read", nil).Code)
	response := requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/?unread=true", nil)
	require.Equal(t, http.StatusOK, response.Code)
	var page struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &page))
	require.Len(t, page.Data, 1)
	require.Equal(t, ids[0], page.Data[0].ID)
	require.Equal(t, http.StatusOK, requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/"+ids[0]+"/save", nil).Code)
	sourceResponse := requestJSON(t, app.Router, http.MethodPost, "/api/v1/sources/", map[string]any{"name": "RSS", "type": "rss", "rssUrl": "https://example.test/feed"})
	require.Equal(t, http.StatusCreated, sourceResponse.Code)
	var source struct {
		ID string `json:"id"`
	}
	decodeResponseData(t, sourceResponse.Body.Bytes(), &source)
	for _, title := range []string{"one", "two"} {
		res := requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/", map[string]any{"sourceId": source.ID, "title": title, "link": "https://example.test/" + title})
		require.Equal(t, http.StatusCreated, res.Code)
		var e struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &e))
		require.Equal(t, http.StatusOK, requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/"+e.Data.ID+"/save", nil).Code)
	}
	type listPage struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Meta struct {
			NextCursor string `json:"nextCursor"`
		} `json:"meta"`
	}
	get := func(query string) listPage {
		res := requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/?"+query, nil)
		require.Equal(t, http.StatusOK, res.Code, res.Body.String())
		var p listPage
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &p))
		return p
	}
	require.Len(t, get("savedOnly=true").Data, 3)
	require.Len(t, get("sourceId="+source.ID).Data, 2)
	require.Len(t, get("sourceId="+source.ID+"&unread=true&savedOnly=true").Data, 2)
	require.Empty(t, get("sourceId=00000000-0000-0000-0000-000000000000").Data)
	first := get("sourceId=" + source.ID + "&savedOnly=true&pageSize=1")
	require.Len(t, first.Data, 1)
	require.NotEmpty(t, first.Meta.NextCursor)
	second := get("sourceId=" + source.ID + "&savedOnly=true&pageSize=1&cursor=" + url.QueryEscape(first.Meta.NextCursor))
	require.Len(t, second.Data, 1)
	require.NotEqual(t, first.Data[0].ID, second.Data[0].ID)
	require.Empty(t, second.Meta.NextCursor)
	for _, query := range []string{"sourceId=bad", "cursor=bad", "unread=maybe", "savedOnly=maybe", "cursor=" + url.QueryEscape(first.Meta.NextCursor)} {
		res := requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/?"+query, nil)
		require.Equal(t, http.StatusBadRequest, res.Code, query)
	}
}

func TestFetchJobHistoryReturnsCursorEnvelopeAndExistingFields(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbPath := filepath.Join(t.TempDir(), "fetch-history.db")
	require.NoError(t, goose.SetDialect("sqlite3"))
	db, err := sql.Open("sqlite", sqliteTestDSN(dbPath))
	require.NoError(t, err)
	require.NoError(t, goose.Up(db, "../../migrations"))
	require.NoError(t, db.Close())
	app, err := NewWithConfig(ctx, Config{DatabasePath: dbPath, HTTPAddr: ":0", Fetch: FetchConfig{RetryBackoff: time.Hour, ScheduleInterval: time.Hour, ExecutorPoll: time.Hour}})
	require.NoError(t, err)
	defer app.Close()

	sourceResponse := requestJSON(t, app.Router, http.MethodPost, "/api/v1/sources/", map[string]any{"name": "History", "type": "rss", "rssUrl": "https://example.test/history.xml"})
	require.Equal(t, http.StatusCreated, sourceResponse.Code, sourceResponse.Body.String())
	var source struct {
		ID string `json:"id"`
	}
	decodeResponseData(t, sourceResponse.Body.Bytes(), &source)
	created := requestJSON(t, app.Router, http.MethodPost, "/api/v1/fetch-jobs/", map[string]any{"sourceId": source.ID})
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	var job struct {
		ID string `json:"id"`
	}
	decodeResponseData(t, created.Body.Bytes(), &job)

	response := requestJSON(t, app.Router, http.MethodGet, "/api/v1/fetch-jobs/", nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var page struct {
		Data []struct {
			ID             string `json:"id"`
			SourceID       string `json:"sourceId"`
			Status         string `json:"status"`
			Attempt        int    `json:"attempt"`
			MaxAttempts    int    `json:"maxAttempts"`
			EntriesCreated int    `json:"entriesCreated"`
			CreatedAt      string `json:"createdAt"`
			UpdatedAt      string `json:"updatedAt"`
		} `json:"data"`
		Meta struct {
			NextCursor string `json:"nextCursor"`
			PageSize   int    `json:"pageSize"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &page))
	require.Len(t, page.Data, 1)
	require.Equal(t, job.ID, page.Data[0].ID)
	require.Equal(t, source.ID, page.Data[0].SourceID)
	require.Equal(t, "pending", page.Data[0].Status)
	require.Equal(t, 0, page.Data[0].Attempt)
	require.Equal(t, 3, page.Data[0].MaxAttempts)
	require.Equal(t, 0, page.Data[0].EntriesCreated)
	require.NotEmpty(t, page.Data[0].CreatedAt)
	require.NotEmpty(t, page.Data[0].UpdatedAt)
	require.Equal(t, 10, page.Meta.PageSize)
	require.Empty(t, page.Meta.NextCursor)
}

func TestFetchJobHistoryFiltersAndContinuesWithStableOrdering(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbPath := filepath.Join(t.TempDir(), "fetch-history-pages.db")
	require.NoError(t, goose.SetDialect("sqlite3"))
	db, err := sql.Open("sqlite", sqliteTestDSN(dbPath))
	require.NoError(t, err)
	require.NoError(t, goose.Up(db, "../../migrations"))
	require.NoError(t, db.Close())
	app, err := NewWithConfig(ctx, Config{DatabasePath: dbPath, HTTPAddr: ":0", Fetch: FetchConfig{RetryBackoff: time.Hour, ScheduleInterval: time.Hour, ExecutorPoll: time.Hour}})
	require.NoError(t, err)
	defer app.Close()

	var jobIDs, sourceIDs []string
	for i := 0; i < 4; i++ {
		sourceResponse := requestJSON(t, app.Router, http.MethodPost, "/api/v1/sources/", map[string]any{"name": fmt.Sprintf("History %d", i), "type": "rss", "rssUrl": fmt.Sprintf("https://example.test/history-%d.xml", i)})
		require.Equal(t, http.StatusCreated, sourceResponse.Code, sourceResponse.Body.String())
		var source struct {
			ID string `json:"id"`
		}
		decodeResponseData(t, sourceResponse.Body.Bytes(), &source)
		sourceIDs = append(sourceIDs, source.ID)
		created := requestJSON(t, app.Router, http.MethodPost, "/api/v1/fetch-jobs/", map[string]any{"sourceId": source.ID})
		require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
		var job struct {
			ID string `json:"id"`
		}
		decodeResponseData(t, created.Body.Bytes(), &job)
		jobIDs = append(jobIDs, job.ID)
	}
	for i, status := range []string{"pending", "running", "succeeded", "failed"} {
		_, err := app.Pool.DB.Exec(`UPDATE fetch_jobs SET status = ?, created_at = ?, updated_at = ? WHERE id = ?`, status, "2025-01-01T00:00:00Z", "2025-01-02T00:00:00Z", jobIDs[i])
		require.NoError(t, err)
	}
	type listPage struct {
		Data []struct {
			ID       string `json:"id"`
			SourceID string `json:"sourceId"`
			Status   string `json:"status"`
		} `json:"data"`
		Meta struct {
			NextCursor string `json:"nextCursor"`
			PageSize   int    `json:"pageSize"`
		} `json:"meta"`
	}
	get := func(query string) (listPage, *httptest.ResponseRecorder) {
		t.Helper()
		response := requestJSON(t, app.Router, http.MethodGet, "/api/v1/fetch-jobs/?"+query, nil)
		var page listPage
		if response.Code == http.StatusOK {
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &page))
		}
		return page, response
	}
	first, response := get("pageSize=1")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Len(t, first.Data, 1)
	require.Equal(t, 1, first.Meta.PageSize)
	require.NotEmpty(t, first.Meta.NextCursor)
	second, response := get("pageSize=3&cursor=" + url.QueryEscape(first.Meta.NextCursor))
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Len(t, second.Data, 3)
	require.Empty(t, second.Meta.NextCursor)
	ordered := []string{first.Data[0].ID, second.Data[0].ID, second.Data[1].ID, second.Data[2].ID}
	expectedOrder := append([]string(nil), jobIDs...)
	sort.Sort(sort.Reverse(sort.StringSlice(expectedOrder)))
	require.Equal(t, expectedOrder, ordered)
	filtered, response := get("sourceId=" + sourceIDs[1] + "&status=running")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Len(t, filtered.Data, 1)
	require.Equal(t, jobIDs[1], filtered.Data[0].ID)
	for i, status := range []string{"pending", "running", "succeeded", "failed"} {
		filtered, response = get("status=" + status)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		require.Len(t, filtered.Data, 1, status)
		require.Equal(t, jobIDs[i], filtered.Data[0].ID, status)
	}
	_, response = get("sourceId=" + sourceIDs[0] + "&cursor=" + url.QueryEscape(first.Meta.NextCursor))
	require.Equal(t, http.StatusBadRequest, response.Code)
	for _, query := range []string{"sourceId=bad", "status=queued", "pageSize=0", "pageSize=101", "pageSize=nope", "cursor=bad"} {
		_, response = get(query)
		require.Equal(t, http.StatusBadRequest, response.Code, query)
	}
	_, response = get("sourceId=00000000-0000-0000-0000-000000000000")
	require.Equal(t, http.StatusOK, response.Code)
	empty, _ := get("sourceId=00000000-0000-0000-0000-000000000000")
	require.Empty(t, empty.Data)
}

func TestFetchJobHistoryOrdersFractionalTimestampsChronologicallyAcrossPages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbPath := filepath.Join(t.TempDir(), "fetch-history-fractions.db")
	require.NoError(t, goose.SetDialect("sqlite3"))
	db, err := sql.Open("sqlite", sqliteTestDSN(dbPath))
	require.NoError(t, err)
	require.NoError(t, goose.Up(db, "../../migrations"))
	require.NoError(t, db.Close())
	app, err := NewWithConfig(ctx, Config{DatabasePath: dbPath, HTTPAddr: ":0", Fetch: FetchConfig{RetryBackoff: time.Hour, ScheduleInterval: time.Hour, ExecutorPoll: time.Hour}})
	require.NoError(t, err)
	defer app.Close()

	var ids []string
	for i := 0; i < 3; i++ {
		sourceResponse := requestJSON(t, app.Router, http.MethodPost, "/api/v1/sources/", map[string]any{"name": fmt.Sprintf("Fraction %d", i), "type": "rss", "rssUrl": fmt.Sprintf("https://example.test/fraction-%d.xml", i)})
		var source struct {
			ID string `json:"id"`
		}
		decodeResponseData(t, sourceResponse.Body.Bytes(), &source)
		created := requestJSON(t, app.Router, http.MethodPost, "/api/v1/fetch-jobs/", map[string]any{"sourceId": source.ID})
		var job struct {
			ID string `json:"id"`
		}
		decodeResponseData(t, created.Body.Bytes(), &job)
		ids = append(ids, job.ID)
	}
	for i, stamp := range []string{"2025-01-01T00:00:00.1Z", "2025-01-01T00:00:00.11Z", "2025-01-01T00:00:00.09Z"} {
		_, err := app.Pool.DB.Exec(`UPDATE fetch_jobs SET created_at = ? WHERE id = ?`, stamp, ids[i])
		require.NoError(t, err)
	}
	type page struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Meta struct {
			NextCursor string `json:"nextCursor"`
			PageSize   int    `json:"pageSize"`
		} `json:"meta"`
	}
	get := func(query string) page {
		t.Helper()
		res := requestJSON(t, app.Router, http.MethodGet, "/api/v1/fetch-jobs/?"+query, nil)
		require.Equal(t, http.StatusOK, res.Code, res.Body.String())
		var result page
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &result))
		return result
	}
	first := get("pageSize=1")
	require.Equal(t, 1, first.Meta.PageSize)
	require.Equal(t, ids[1], first.Data[0].ID)
	require.NotEmpty(t, first.Meta.NextCursor)
	second := get("pageSize=1&cursor=" + url.QueryEscape(first.Meta.NextCursor))
	require.Equal(t, ids[0], second.Data[0].ID)
	require.NotEmpty(t, second.Meta.NextCursor)
	third := get("pageSize=1&cursor=" + url.QueryEscape(second.Meta.NextCursor))
	require.Equal(t, ids[2], third.Data[0].ID)
	require.Empty(t, third.Meta.NextCursor)
}

func TestFetchJobHistoryUsesDefaultAndMaximumPageSizes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbPath := filepath.Join(t.TempDir(), "fetch-history-page-sizes.db")
	require.NoError(t, goose.SetDialect("sqlite3"))
	db, err := sql.Open("sqlite", sqliteTestDSN(dbPath))
	require.NoError(t, err)
	require.NoError(t, goose.Up(db, "../../migrations"))
	require.NoError(t, db.Close())
	app, err := NewWithConfig(ctx, Config{DatabasePath: dbPath, HTTPAddr: ":0", Fetch: FetchConfig{RetryBackoff: time.Hour, ScheduleInterval: time.Hour, ExecutorPoll: time.Hour}})
	require.NoError(t, err)
	defer app.Close()

	for i := 0; i < 101; i++ {
		sourceResponse := requestJSON(t, app.Router, http.MethodPost, "/api/v1/sources/", map[string]any{"name": fmt.Sprintf("Page size %d", i), "type": "rss", "rssUrl": fmt.Sprintf("https://example.test/page-size-%d.xml", i)})
		require.Equal(t, http.StatusCreated, sourceResponse.Code, sourceResponse.Body.String())
		var source struct {
			ID string `json:"id"`
		}
		decodeResponseData(t, sourceResponse.Body.Bytes(), &source)
		created := requestJSON(t, app.Router, http.MethodPost, "/api/v1/fetch-jobs/", map[string]any{"sourceId": source.ID})
		require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	}
	for _, tc := range []struct {
		query string
		want  int
	}{
		{query: "", want: 10},
		{query: "pageSize=100", want: 100},
	} {
		res := requestJSON(t, app.Router, http.MethodGet, "/api/v1/fetch-jobs/?"+tc.query, nil)
		require.Equal(t, http.StatusOK, res.Code, res.Body.String())
		var page struct {
			Data []json.RawMessage `json:"data"`
			Meta struct {
				NextCursor string `json:"nextCursor"`
				PageSize   int    `json:"pageSize"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &page))
		require.Len(t, page.Data, tc.want)
		require.Equal(t, tc.want, page.Meta.PageSize)
		require.NotEmpty(t, page.Meta.NextCursor)
	}
}

func TestSavedCountEndpointTracksSavedEntriesAcrossPages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbPath := filepath.Join(t.TempDir(), "saved-count.db")
	require.NoError(t, goose.SetDialect("sqlite3"))
	db, err := sql.Open("sqlite", sqliteTestDSN(dbPath))
	require.NoError(t, err)
	require.NoError(t, goose.Up(db, "../../migrations"))
	require.NoError(t, db.Close())
	cfg := Config{DatabasePath: dbPath, HTTPAddr: ":0", Fetch: FetchConfig{RetryBackoff: time.Hour, ScheduleInterval: time.Hour, ExecutorPoll: time.Hour}}
	app, err := NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	defer func() { app.Close() }()

	count := func() int {
		response := requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/saved-count", nil)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var result struct {
			Data struct {
				Count int `json:"count"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		return result.Data.Count
	}
	require.Equal(t, 0, count())

	entryIDs := make([]string, 21)
	for i := range entryIDs {
		created := requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/save-link", map[string]any{
			"title": fmt.Sprintf("Entry %d", i), "link": fmt.Sprintf("https://example.test/%d", i),
		})
		require.Equal(t, http.StatusCreated, created.Code)
		var entry struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(created.Body.Bytes(), &entry))
		entryIDs[i] = entry.Data.ID
		require.Equal(t, http.StatusOK, requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/"+entry.Data.ID+"/save", nil).Code)
	}
	require.Equal(t, 21, count())
	page := requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/?savedOnly=true", nil)
	require.Equal(t, http.StatusOK, page.Code)
	var result struct {
		Data []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(page.Body.Bytes(), &result))
	require.Len(t, result.Data, 10)
	require.Equal(t, http.StatusOK, requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/"+entryIDs[0]+"/clear-save", nil).Code)
	require.Equal(t, 20, count())
	require.NoError(t, app.Pool.Close())
	app, err = NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	require.Equal(t, 20, count())
}

func TestSQLiteAtomFetchLifecycle(t *testing.T) {
	for _, scenario := range []string{"refresh preserves reader state", "empty document succeeds", "unsupported document fails"} {
		t.Run(scenario, func(t *testing.T) {
			// Arrange
			var mu sync.RWMutex
			body := `<rss><channel><item><title>Original</title><link>https://example.test/page</link></item></channel></rss>`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				mu.RLock()
				defer mu.RUnlock()
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()
			setBody := func(value string) { mu.Lock(); defer mu.Unlock(); body = value }
			ctx, cancel := context.WithCancel(context.Background())
			dbPath := filepath.Join(t.TempDir(), "atom.db")
			require.NoError(t, goose.SetDialect("sqlite3"))
			db, err := sql.Open("sqlite", sqliteTestDSN(dbPath))
			require.NoError(t, err)
			require.NoError(t, goose.Up(db, "../../migrations"))
			require.NoError(t, db.Close())
			cfg := Config{DatabasePath: dbPath, HTTPAddr: ":0", Fetch: FetchConfig{RetryBackoff: time.Millisecond, ScheduleInterval: time.Hour, ExecutorPoll: 5 * time.Millisecond}}
			setupCtx, setupCancel := context.WithCancel(context.Background())
			app, err := NewWithConfig(setupCtx, cfg)
			require.NoError(t, err)
			setupCancel()

			response := requestJSON(t, app.Router, http.MethodPost, "/api/v1/sources/", map[string]any{"name": "Atom", "type": "rss", "rssUrl": server.URL})
			require.Equal(t, http.StatusCreated, response.Code)
			var source struct {
				ID string `json:"id"`
			}
			decodeResponseData(t, response.Body.Bytes(), &source)
			app.Close()
			app, err = NewWithConfig(ctx, cfg)
			require.NoError(t, err)
			defer func() { cancel(); app.Close() }()
			var scheduled struct {
				Data []struct {
					ID             string `json:"id"`
					Status         string `json:"status"`
					EntriesCreated int    `json:"entriesCreated"`
				} `json:"data"`
			}
			require.Eventually(t, func() bool {
				res := requestJSON(t, app.Router, http.MethodGet, "/api/v1/fetch-jobs/?sourceId="+source.ID, nil)
				require.NoError(t, json.Unmarshal(res.Body.Bytes(), &scheduled))
				return len(scheduled.Data) == 1 && scheduled.Data[0].Status == "succeeded"
			}, 5*time.Second, 10*time.Millisecond)
			require.Equal(t, 1, scheduled.Data[0].EntriesCreated)
			run := func(status string, count int) {
				t.Helper()
				res := requestJSON(t, app.Router, http.MethodPost, "/api/v1/fetch-jobs/", map[string]any{"sourceId": source.ID})
				require.Equal(t, http.StatusCreated, res.Code, res.Body.String())
				var job struct {
					ID             string `json:"id"`
					Status         string `json:"status"`
					EntriesCreated int    `json:"entriesCreated"`
					Attempt        int    `json:"attempt"`
				}
				decodeResponseData(t, res.Body.Bytes(), &job)
				require.Eventually(t, func() bool {
					res := requestJSON(t, app.Router, http.MethodGet, "/api/v1/fetch-jobs/"+job.ID, nil)
					require.Equal(t, http.StatusOK, res.Code, res.Body.String())
					decodeResponseData(t, res.Body.Bytes(), &job)
					return job.Status == status
				}, 5*time.Second, 10*time.Millisecond)
				require.Equal(t, count, job.EntriesCreated)
				if status == "failed" {
					require.Equal(t, 3, job.Attempt)
				}
			}
			if scenario == "empty document succeeds" {
				setBody(`<feed xmlns="http://www.w3.org/2005/Atom"/>`)
				// Act & Assert
				run("succeeded", 0)
				return
			}
			if scenario == "unsupported document fails" {
				setBody(`<html/>`)
				// Act & Assert
				run("failed", 0)
				return
			}
			list := requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/", nil)
			var entries struct {
				Data []struct {
					ID          string  `json:"id"`
					Title       string  `json:"title"`
					Description string  `json:"description"`
					ReadAt      *string `json:"readAt"`
					SavedAt     *string `json:"savedAt"`
					Thumbnail   *string `json:"thumbnail"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(list.Body.Bytes(), &entries))
			require.Len(t, entries.Data, 1)
			id := entries.Data[0].ID
			require.Equal(t, http.StatusOK, requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/"+id+"/read", nil).Code)
			require.Equal(t, http.StatusOK, requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/"+id+"/save", nil).Code)
			setBody(`<feed xmlns="http://www.w3.org/2005/Atom" xmlns:media="http://search.yahoo.com/mrss/"><entry><title>Refreshed</title><link href="https://example.test/page"/><link rel="enclosure" type="image/png" href="/fallback.png"/><media:thumbnail url="/selected.png"/><summary type="html">&lt;p&gt;Read &lt;strong&gt;this&lt;/strong&gt;&lt;/p&gt;</summary></entry></feed>`)

			// Act
			run("succeeded", 0)
			// Assert
			list = requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/", nil)
			require.NoError(t, json.Unmarshal(list.Body.Bytes(), &entries))
			require.Len(t, entries.Data, 1)
			require.Equal(t, id, entries.Data[0].ID)
			require.Equal(t, "Refreshed", entries.Data[0].Title)
			require.Equal(t, "<p>Read <strong>this</strong></p>", entries.Data[0].Description)
			require.NotNil(t, entries.Data[0].Thumbnail)
			require.Equal(t, "/fallback.png", *entries.Data[0].Thumbnail)

			require.NotNil(t, entries.Data[0].ReadAt)
			require.NotNil(t, entries.Data[0].SavedAt)
		})
	}
}

func decodeResponseData(t *testing.T, body []byte, dst any) {
	t.Helper()
	var envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &envelope))
	require.True(t, envelope.Success)
	require.NoError(t, json.Unmarshal(envelope.Data, dst))
}
