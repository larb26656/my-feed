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
	dbPath := filepath.Join(t.TempDir(), "odd # name? with spaces", "quick-feed.db")
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
	require.NoError(t, json.Unmarshal(createSource.Body.Bytes(), &source))

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
	require.NoError(t, json.Unmarshal(createJob.Body.Bytes(), &job))
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
	dbPath := filepath.Join(t.TempDir(), "quick-feed.db")
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
	require.NoError(t, json.Unmarshal(createSource.Body.Bytes(), &source))
	createJob := requestJSON(t, app.Router, http.MethodPost, "/api/v1/fetch-jobs/", map[string]any{"sourceId": source.ID})
	require.Equal(t, http.StatusCreated, createJob.Code)
	var job struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(createJob.Body.Bytes(), &job))

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
	dbPath := filepath.Join(t.TempDir(), "quick-feed.db")
	backupPath := filepath.Join(t.TempDir(), "quick-feed-backup.db")
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
	require.NoError(t, json.Unmarshal(createSource.Body.Bytes(), &source))
	createEntry := requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/save-link", map[string]any{
		"title": "Backup entry", "link": "https://example.test/backup-entry",
	})
	require.Equal(t, http.StatusCreated, createEntry.Code)
	var entry struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(createEntry.Body.Bytes(), &entry))
	require.Equal(t, http.StatusOK, requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/"+entry.ID+"/read", map[string]any{}).Code)
	require.Equal(t, http.StatusOK, requestJSON(t, app.Router, http.MethodPost, "/api/v1/entries/"+entry.ID+"/save", map[string]any{}).Code)
	createJob := requestJSON(t, app.Router, http.MethodPost, "/api/v1/fetch-jobs/", map[string]any{"sourceId": source.ID})
	require.Equal(t, http.StatusCreated, createJob.Code)
	var job struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(createJob.Body.Bytes(), &job))

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
	getEntry := requestJSON(t, app.Router, http.MethodGet, "/api/v1/entries/"+entry.ID, nil)
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
