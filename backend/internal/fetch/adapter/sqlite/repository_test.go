package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/larb26656/my-feed/backend/internal/fetch/domain"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

func openTestRepository(t *testing.T) (*sql.DB, *Repository) {
	t.Helper()
	dbPath := (&url.URL{Scheme: "file", Path: filepath.Join(t.TempDir(), "fetch-jobs.db")}).String()
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	db.SetMaxOpenConns(8)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(`CREATE TABLE sources (id TEXT PRIMARY KEY);`)
	require.NoError(t, err)
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrationPath := filepath.Join(filepath.Dir(currentFile), "../../../../migrations/000002_fetch_jobs.sql")
	migration, err := os.ReadFile(migrationPath)
	require.NoError(t, err)
	upSection := strings.SplitN(strings.TrimPrefix(string(migration), "-- +goose Up\n"), "-- +goose Down", 2)[0]
	_, err = db.Exec(upSection)
	require.NoError(t, err)
	return db, NewRepository(db)
}

func newTestJob(sourceID uuid.UUID, at time.Time) *domain.FetchJob {
	return &domain.FetchJob{ID: uuid.New(), SourceID: sourceID, Status: domain.StatusPending, MaxAttempts: 3, CreatedAt: at, UpdatedAt: at}
}

func addTestSource(t *testing.T, db *sql.DB, id uuid.UUID) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO sources(id) VALUES (?)`, id.String())
	require.NoError(t, err)
}

func TestFetchJobsPersistLifecycleAndPermitNewJobAfterTerminalOutcome(t *testing.T) {
	db, repo := openTestRepository(t)
	ctx := context.Background()
	sourceID := uuid.New()
	addTestSource(t, db, sourceID)
	now := time.Date(2026, 11, 15, 4, 5, 6, 123456789, time.UTC)
	job := newTestJob(sourceID, now)
	require.NoError(t, repo.CreateFetchJob(ctx, job))

	claimed, err := repo.ClaimFetchJob(ctx, job.ID, now)
	require.NoError(t, err)
	require.Equal(t, domain.StatusRunning, claimed.Status)
	require.Equal(t, 1, claimed.Attempt)
	claimed.Status, claimed.EntriesCreated, claimed.UpdatedAt = domain.StatusSucceeded, 2, now.Add(time.Second)
	require.NoError(t, repo.UpdateFetchJob(ctx, claimed))

	got, err := repo.GetFetchJobByID(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusSucceeded, got.Status)
	require.Equal(t, 2, got.EntriesCreated)
	require.Equal(t, now, got.CreatedAt)
	require.Equal(t, now.Add(time.Second), got.UpdatedAt)

	second := newTestJob(sourceID, now.Add(2*time.Second))
	require.NoError(t, repo.CreateFetchJob(ctx, second), "terminal jobs must not block a later Fetch Job")
}

func TestFetchJobsAllowOnlyOneUnfinishedJobPerSourceUnderConcurrentCreation(t *testing.T) {
	db, repo := openTestRepository(t)
	ctx := context.Background()
	sourceID := uuid.New()
	addTestSource(t, db, sourceID)
	const contenders = 8
	start := make(chan struct{})
	errs := make(chan error, contenders)
	var wg sync.WaitGroup
	for range contenders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- repo.CreateFetchJob(ctx, newTestJob(sourceID, time.Now().UTC()))
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	created, conflicts := 0, 0
	for err := range errs {
		if err == nil {
			created++
			continue
		}
		var appErr *apperr.Error
		if errors.As(err, &appErr) && appErr.Code == apperr.CodeConflict {
			conflicts++
			continue
		}
		require.NoError(t, err)
	}
	require.Equal(t, 1, created)
	require.Equal(t, contenders-1, conflicts)
	var count int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fetch_jobs WHERE status IN ('pending','running')`).Scan(&count))
	require.Equal(t, 1, count)
}

func TestFetchJobHasOnlyOneWinnerWhenClaimedConcurrently(t *testing.T) {
	ctx := context.Background()
	sourceID := uuid.New()
	job := newTestJob(sourceID, time.Now().UTC())
	db, repo := openTestRepository(t)
	addTestSource(t, db, sourceID)
	require.NoError(t, repo.CreateFetchJob(ctx, job))
	const contenders = 8
	start := make(chan struct{})
	errs := make(chan error, contenders)
	var wg sync.WaitGroup
	for range contenders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := repo.ClaimFetchJob(ctx, job.ID, time.Now().UTC())
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	wins, rejected := 0, 0
	for err := range errs {
		if err == nil {
			wins++
			continue
		}
		var appErr *apperr.Error
		if errors.As(err, &appErr) && appErr.Code == apperr.CodeFailedPrecondition {
			rejected++
			continue
		}
		require.NoError(t, err)
	}
	require.Equal(t, 1, wins)
	require.Equal(t, contenders-1, rejected)
	got, err := repo.GetFetchJobByID(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, 1, got.Attempt)
	require.Equal(t, domain.StatusRunning, got.Status)
}

func TestClaimableFetchJobsRespectRetryBackoff(t *testing.T) {
	db, repo := openTestRepository(t)
	ctx := context.Background()
	sourceID := uuid.New()
	addTestSource(t, db, sourceID)
	now := time.Date(2026, 11, 15, 4, 0, 0, 0, time.UTC)
	job := newTestJob(sourceID, now)
	job.NextAttemptAt = now.Add(time.Minute)
	require.NoError(t, repo.CreateFetchJob(ctx, job))

	ready, err := repo.ListClaimableFetchJobs(ctx, now.Add(59*time.Second))
	require.NoError(t, err)
	require.Empty(t, ready)
	ready, err = repo.ListClaimableFetchJobs(ctx, now.Add(time.Minute))
	require.NoError(t, err)
	require.Len(t, ready, 1)
	_, err = repo.ClaimFetchJob(ctx, job.ID, now.Add(59*time.Second))
	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperr.CodeFailedPrecondition, appErr.Code)
}
