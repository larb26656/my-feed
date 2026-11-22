package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/source/repository"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestSQLiteRetrievesTheManualSource(t *testing.T) {
	// Arrange
	pool := testkit.NewDatabase(t)
	repo := NewRepository(pool.DB)
	manual, err := domain.NewManualSource(domain.SavedLinksName)
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(context.Background(), manual))

	// Act
	got, err := repo.GetManualSource(context.Background())

	// Assert
	require.NoError(t, err)
	require.Equal(t, manual.ID, got.ID)
	require.Equal(t, manual.CreatedAt.UnixNano(), got.CreatedAt.UnixNano())
}

func TestSQLiteAllowsOnlyOneManualSource(t *testing.T) {
	// Arrange
	pool := testkit.NewDatabase(t)
	repo := NewRepository(pool.DB)
	first, err := domain.NewManualSource(domain.SavedLinksName)
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(context.Background(), first))
	duplicate, err := domain.NewManualSource(domain.SavedLinksName)
	require.NoError(t, err)

	// Act
	err = repo.CreateSource(context.Background(), duplicate)

	// Assert
	var failure *apperr.AppError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, apperr.ResourceConflict.Code, failure.Code)
}

func TestSQLiteRejectsDuplicateFeedURLs(t *testing.T) {
	// Arrange
	pool := testkit.NewDatabase(t)
	repo := NewRepository(pool.DB)
	first, err := domain.NewRSSSource("News", "https://example.test/rss")
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(context.Background(), first))
	duplicate, err := domain.NewRSSSource("Another name", first.RSSURL)
	require.NoError(t, err)

	// Act
	err = repo.CreateSource(context.Background(), duplicate)

	// Assert
	var failure *apperr.AppError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, apperr.ResourceConflict.Code, failure.Code)
}

func TestSQLiteSourceListsRespectTheIncludeDeactivatedFilter(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		includePaused bool
		count         int
	}{
		{"active sources only", false, 1}, {"including paused sources", true, 2},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Arrange
			pool := testkit.NewDatabase(t)
			repo := NewRepository(pool.DB)
			manual, err := domain.NewManualSource(domain.SavedLinksName)
			require.NoError(t, err)
			require.NoError(t, repo.CreateSource(context.Background(), manual))
			paused, err := domain.NewRSSSource("News", "https://example.test/rss")
			require.NoError(t, err)
			require.NoError(t, paused.Deactivate())
			require.NoError(t, repo.CreateSource(context.Background(), paused))

			// Act
			sources, err := repo.ListSources(context.Background(), repository.ListSourcesFilter{IncludeDeactivated: scenario.includePaused})

			// Assert
			require.NoError(t, err)
			require.Len(t, sources, scenario.count)
			if !scenario.includePaused {
				require.Equal(t, manual.ID, sources[0].ID)
			}
		})
	}
}

func TestSQLiteSourceStateSurvivesClosingAndReopeningTheDatabase(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "sources.db")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	testkit.Migrate(t, db)
	repo := NewRepository(db)
	source, err := domain.NewRSSSource("News", "https://example.test/rss")
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(context.Background(), source))
	require.NoError(t, source.Deactivate())
	require.NoError(t, repo.UpdateSource(context.Background(), source))
	require.NoError(t, db.Close())

	// Act
	db, err = sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	got, err := NewRepository(db).GetSourceByID(context.Background(), source.ID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, source.ID, got.ID)
	require.False(t, got.Active)
	require.Equal(t, source.UpdatedAt.UnixNano(), got.UpdatedAt.UnixNano())
}

func TestSQLiteReturnsNotFoundForAMissingSource(t *testing.T) {
	// Arrange
	pool := testkit.NewDatabase(t)
	repo := NewRepository(pool.DB)

	// Act
	_, err := repo.GetSourceByID(context.Background(), uuid.New())

	// Assert
	var failure *apperr.AppError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, apperr.ResourceNotFound.Code, failure.Code)
}

func TestSQLiteReadsStoredRSSConfig(t *testing.T) {
	pool := testkit.NewDatabase(t)
	repo := NewRepository(pool.DB)
	src, err := domain.NewRSSSource("Feed", "https://old.example/feed")
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(context.Background(), src))
	_, err = pool.Exec(`UPDATE sources SET config = ? WHERE id = ?`, `{"url":"https://new.example/feed"}`, src.ID.String())
	require.NoError(t, err)
	got, err := repo.GetSourceByID(context.Background(), src.ID)
	require.NoError(t, err)
	require.Equal(t, "https://new.example/feed", got.RSSURL)
}

func TestSQLiteStoresRegisteredTypeConfigAndEnforcesPerTypeKey(t *testing.T) {
	pool := testkit.NewDatabase(t)
	repo := NewRepository(pool.DB)
	ctx := context.Background()
	now := time.Now().UTC()
	first := &domain.Source{ID: uuid.New(), Name: "Custom", Type: "custom", Config: map[string]any{"slug": "same", "limit": float64(5)}, Key: "same", Active: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, repo.CreateSource(ctx, first))
	got, err := repo.GetSourceByID(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, first.Config, got.Config)
	require.Equal(t, first.Key, got.Key)
	duplicate := *first
	duplicate.ID = uuid.New()
	require.Error(t, repo.CreateSource(ctx, &duplicate))
	duplicate.Type = "other"
	require.NoError(t, repo.CreateSource(ctx, &duplicate))
}
