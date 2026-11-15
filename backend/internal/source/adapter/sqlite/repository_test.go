package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/source/domain"
	"github.com/larb26656/my-feed/backend/internal/source/repository"
)

func TestSourceRepositoryPersistsSourcesAndEnforcesConstraints(t *testing.T) {
	// Arrange
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "feed.db")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	testkit.Migrate(t, db)
	repo := NewRepository(db)

	manual, err := domain.NewManualSource(domain.SavedLinksName)
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(ctx, manual))

	// Act: retrieve the persisted manual source.
	got, err := repo.GetManualSource(ctx)

	// Assert
	require.NoError(t, err)
	require.Equal(t, manual.ID, got.ID)
	require.Equal(t, manual.CreatedAt.UnixNano(), got.CreatedAt.UnixNano())

	// Arrange: another candidate for the manual singleton.
	duplicateManual, err := domain.NewManualSource(domain.SavedLinksName)
	require.NoError(t, err)

	// Act: create another manual source.
	err = repo.CreateSource(ctx, duplicateManual)

	// Assert
	require.Error(t, err)
	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperr.CodeConflict, appErr.Code)

	// Arrange: two RSS sources sharing one feed URL.
	source, err := domain.NewRSSSource("Example", "https://example.test/feed")
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(ctx, source))
	duplicate, err := domain.NewRSSSource("Duplicate", source.RSSURL)
	require.NoError(t, err)

	// Act: create another source with the same feed URL.
	err = repo.CreateSource(ctx, duplicate)

	// Assert
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperr.CodeConflict, appErr.Code)

	// Act: pause the source and list active sources.
	require.NoError(t, source.Deactivate())

	// Assert
	require.NoError(t, repo.UpdateSource(ctx, source))
	active, err := repo.ListSources(ctx, repository.ListSourcesFilter{})
	require.NoError(t, err)
	require.Len(t, active, 1)

	// Act: include paused sources.
	all, err := repo.ListSources(ctx, repository.ListSourcesFilter{IncludeDeactivated: true})

	// Assert
	require.NoError(t, err)
	require.Len(t, all, 2)

	// Arrange: reopen the same persisted database.
	require.NoError(t, db.Close())
	db, err = sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	defer db.Close()
	repo = NewRepository(db)

	// Act: retrieve the paused source after reopening.
	got, err = repo.GetSourceByID(ctx, source.ID)

	// Assert
	require.NoError(t, err)
	require.False(t, got.Active)
	require.Equal(t, source.UpdatedAt.UnixNano(), got.UpdatedAt.UnixNano())

	badID := uuid.New()

	// Act: look up a missing source.
	_, err = repo.GetSourceByID(ctx, badID)

	// Assert
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperr.CodeNotFound, appErr.Code)
}
