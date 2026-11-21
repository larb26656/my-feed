package bootstrap

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	entrysqlite "github.com/larb26656/my-feed/backend/internal/entry/adapter/sqlite"
	entrysvc "github.com/larb26656/my-feed/backend/internal/entry/service"
	fetchsqlite "github.com/larb26656/my-feed/backend/internal/fetch/adapter/sqlite"
	fetchsvc "github.com/larb26656/my-feed/backend/internal/fetch/service"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/source/adapter/sqlite"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/source/service"
	"github.com/larb26656/my-feed/backend/internal/testkit"
)

func TestYouTubeMigrationPreservesExistingRelationships(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db, err := database.NewPool(ctx, filepath.Join(t.TempDir(), "upgrade.db"))
	require.NoError(t, err)
	defer db.Close()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db.DB, os.DirFS("../../migrations"))
	require.NoError(t, err)
	_, err = provider.UpTo(ctx, 2)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO sources VALUES ('rss','RSS','rss','https://example.com',1,1,1), ('manual','Saved links','manual',NULL,1,1,1); INSERT INTO entries(id,source_id,title,link,created_at,updated_at) VALUES ('entry','rss','Entry','https://example.com/entry',1,1); INSERT INTO fetch_jobs(id,source_id,status) VALUES ('job','rss','succeeded')`)
	require.NoError(t, err)
	// Act
	_, err = provider.Up(ctx)
	require.NoError(t, err)
	// Assert
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM entries e JOIN sources s ON e.source_id=s.id JOIN fetch_jobs f ON f.source_id=s.id WHERE e.id='entry' AND f.id='job'`).Scan(&count))
	require.Equal(t, 1, count)
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	require.NoError(t, err)
	defer rows.Close()
	require.False(t, rows.Next())
}

func TestCreateYouTubeSourceQueuesInitialJob(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbPath := filepath.Join(t.TempDir(), "app.db")
	db, err := database.NewPool(ctx, dbPath)
	require.NoError(t, err)
	provider, err := goose.NewProvider(goose.DialectSQLite3, db.DB, os.DirFS("../../migrations"))
	require.NoError(t, err)
	_, err = provider.Up(ctx)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	app, err := NewWithConfig(ctx, Config{DatabasePath: dbPath, Fetch: FetchConfig{RetryBackoff: time.Hour, ScheduleInterval: time.Hour, ExecutorPoll: time.Hour}})
	require.NoError(t, err)
	defer app.Close()
	// Act
	response := requestJSON(t, app.Router, http.MethodPost, "/api/v1/sources/", map[string]any{"name": " Channel ", "type": "youtube", "youtubeChannelId": "UC1234567890123456789012"})
	// Assert
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var count int
	require.NoError(t, app.Pool.QueryRowContext(ctx, `SELECT count(*) FROM sources s JOIN fetch_jobs j ON j.source_id=s.id WHERE s.youtube_channel_id=?`, "UC1234567890123456789012").Scan(&count))
	require.Equal(t, 1, count)

}

func TestCreateYouTubeRollsBackIfInitialJobCannotBeQueued(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbPath := filepath.Join(t.TempDir(), "app.db")
	db, err := database.NewPool(ctx, dbPath)
	require.NoError(t, err)
	provider, err := goose.NewProvider(goose.DialectSQLite3, db.DB, os.DirFS("../../migrations"))
	require.NoError(t, err)
	_, err = provider.Up(ctx)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	app, err := NewWithConfig(ctx, Config{DatabasePath: dbPath, Fetch: FetchConfig{RetryBackoff: time.Hour, ScheduleInterval: time.Hour, ExecutorPoll: time.Hour}})
	require.NoError(t, err)
	defer app.Close()
	_, err = app.Pool.ExecContext(ctx, `CREATE TRIGGER fail_job BEFORE INSERT ON fetch_jobs BEGIN SELECT RAISE(ABORT, 'queue unavailable'); END`)
	require.NoError(t, err)
	// Act
	response := requestJSON(t, app.Router, http.MethodPost, "/api/v1/sources/", map[string]any{"name": "Channel", "type": "youtube", "youtubeChannelId": "UC1234567890123456789012"})
	// Assert
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Contains(t, response.Body.String(), "INTERNAL_SERVER_ERROR")
	require.NotContains(t, response.Body.String(), "queue unavailable")
	var count int
	require.NoError(t, app.Pool.QueryRowContext(ctx, `SELECT count(*) FROM sources WHERE type='youtube'`).Scan(&count))
	require.Zero(t, count)
}

type youtubeTransport func(*http.Request) (*http.Response, error)

func (f youtubeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestProductionYouTubeRegistryObtainsAndIngestsInitialJob(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db := testkit.NewDatabase(t)
	sourceRepo := sourcesqlite.NewRepository(db.DB)
	fetchRepo := fetchsqlite.NewRepository(db.DB)
	sourceService := sourcesvc.New(sourceRepo)
	entryService := entrysvc.New(entrysqlite.NewRepository(db.DB), sourceLookup{repo: sourceRepo})
	client := &http.Client{Transport: youtubeTransport(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "https://www.youtube.com/feeds/videos.xml?channel_id=UC1234567890123456789012", r.URL.String())
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`<feed xmlns="http://www.w3.org/2005/Atom" xmlns:yt="http://www.youtube.com/xml/schemas/2015"><entry><yt:videoId>video123</yt:videoId><title>Video</title></entry></feed>`))}, nil
	})}
	fetchService := fetchsvc.New(fetchRepo, sourceService, newObtainRegistry(client), entrysvc.NewIngester(entryService), realClock{}, time.Minute, time.Hour)
	api := sourceAPIWithFetch{db: db, sourceRepo: sourceRepo, fetchRepo: fetchRepo, inner: sourceService, fetch: fetchService}
	src, err := api.CreateSource(ctx, sourcesvc.CreateSourceInput{Name: "Channel", Type: "youtube", YouTubeChannelID: "UC1234567890123456789012"})
	require.NoError(t, err)
	// Act
	require.NoError(t, fetchService.ExecuteReadyFetchJobs(ctx))
	// Assert
	var title, link, status string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT e.title,e.link,j.status FROM entries e JOIN fetch_jobs j ON e.source_id=j.source_id WHERE e.source_id=?`, src.ID.String()).Scan(&title, &link, &status))
	require.Equal(t, "Video", title)
	require.Equal(t, "https://www.youtube.com/watch?v=video123", link)
	require.Equal(t, "succeeded", status)
}

type controlledClock struct{ now time.Time }

func (c *controlledClock) Now() time.Time { return c.now }

func TestYouTubeRetryRefreshAndOmissionPreserveEntries(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db := testkit.NewDatabase(t)
	sourceRepo := sourcesqlite.NewRepository(db.DB)
	fetchRepo := fetchsqlite.NewRepository(db.DB)
	sourceService := sourcesvc.New(sourceRepo)
	entryService := entrysvc.New(entrysqlite.NewRepository(db.DB), sourceLookup{repo: sourceRepo})
	body := `<feed xmlns="http://www.w3.org/2005/Atom" xmlns:yt="http://www.youtube.com/xml/schemas/2015"><entry><yt:videoId>first</yt:videoId><title>Original</title></entry><entry><yt:videoId>second</yt:videoId><title>Second</title></entry></feed>`
	client := &http.Client{Transport: youtubeTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	clock := &controlledClock{now: time.Now().UTC()}
	fetchService := fetchsvc.New(fetchRepo, sourceService, newObtainRegistry(client), entrysvc.NewIngester(entryService), clock, time.Minute, time.Hour)
	api := sourceAPIWithFetch{db: db, sourceRepo: sourceRepo, fetchRepo: fetchRepo, inner: sourceService, fetch: fetchService}
	src, err := api.CreateSource(ctx, sourcesvc.CreateSourceInput{Name: "Channel", Type: "youtube", YouTubeChannelID: "UC1234567890123456789012"})
	require.NoError(t, err)
	job, err := fetchRepo.GetUnfinishedFetchJobBySourceID(ctx, src.ID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE TRIGGER fail_second BEFORE INSERT ON entries WHEN NEW.link = 'https://www.youtube.com/watch?v=second' BEGIN SELECT RAISE(ABORT, 'temporary failure'); END`)
	require.NoError(t, err)
	// Arrange: establish a partial ingestion and reader choices for the retry.
	failed, err := fetchService.ExecuteFetchJob(ctx, job.ID)
	require.Error(t, err)
	require.Equal(t, "pending", string(failed.Status))
	require.Equal(t, 1, failed.Attempt)
	require.Equal(t, clock.now.Add(time.Minute), failed.NextAttemptAt)
	list, err := entryService.ListEntries(ctx, entrysvc.ListEntriesInput{})
	require.NoError(t, err)
	require.Len(t, list.Entries, 1)
	first := list.Entries[0]
	read, err := entryService.MarkEntryRead(ctx, first.ID)
	require.NoError(t, err)
	saved, err := entryService.MarkEntrySaved(ctx, first.ID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `DROP TRIGGER fail_second`)
	require.NoError(t, err)
	body = strings.Replace(body, "Original", "Updated", 1)
	clock.now = clock.now.Add(time.Minute)
	// Act
	succeeded, err := fetchService.ExecuteFetchJob(ctx, job.ID)
	require.NoError(t, err)
	// Assert
	require.Equal(t, "succeeded", string(succeeded.Status))
	require.Equal(t, 2, succeeded.Attempt)
	require.Equal(t, 1, succeeded.EntriesCreated)
	refreshed, err := entryService.GetEntry(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, "Updated", refreshed.Title)
	require.Equal(t, read.ReadAt, refreshed.ReadAt)
	require.Equal(t, saved.SavedAt, refreshed.SavedAt)
	list, err = entryService.ListEntries(ctx, entrysvc.ListEntriesInput{})
	require.NoError(t, err)
	require.Len(t, list.Entries, 2)
	body = `<feed xmlns="http://www.w3.org/2005/Atom" xmlns:yt="http://www.youtube.com/xml/schemas/2015"><entry><yt:videoId>first</yt:videoId><title>Updated again</title></entry></feed>`
	next, err := fetchService.CreateManualFetchJob(ctx, src.ID)
	require.NoError(t, err)
	next, err = fetchService.ExecuteFetchJob(ctx, next.ID)
	require.NoError(t, err)
	require.Zero(t, next.EntriesCreated)
	list, err = entryService.ListEntries(ctx, entrysvc.ListEntriesInput{})
	require.NoError(t, err)
	require.Len(t, list.Entries, 2)
}

func TestYouTubeMigrationEnforcesChannelIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, channel string
		duplicate     bool
	}{
		{"duplicate inactive channel", "UC1234567890123456789012", true},
		{"invalid channel", "invalid", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()
			db := testkit.NewDatabase(t)
			if tc.duplicate {
				_, err := db.ExecContext(ctx, `INSERT INTO sources(id,name,type,youtube_channel_id,active,created_at,updated_at) VALUES ('yt','YouTube','youtube','UC1234567890123456789012',0,1,1)`)
				require.NoError(t, err)
			}
			// Act
			_, err := db.ExecContext(ctx, `INSERT INTO sources(id,name,type,youtube_channel_id,active,created_at,updated_at) VALUES ('candidate','YouTube','youtube',?,1,1,1)`, tc.channel)
			// Assert
			require.Error(t, err)
		})
	}
}
