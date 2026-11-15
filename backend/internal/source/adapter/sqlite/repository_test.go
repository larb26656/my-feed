package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/source/domain"
	"github.com/larb26656/my-feed/backend/internal/source/repository"
)

func TestSourceRepositoryPersistsSourcesAndEnforcesConstraints(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "feed.db")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	applySourcesMigration(t, db)
	repo := NewRepository(db)

	manual, err := domain.NewManualSource(domain.SavedLinksName)
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(ctx, manual))
	got, err := repo.GetManualSource(ctx)
	require.NoError(t, err)
	require.Equal(t, manual.ID, got.ID)
	require.Equal(t, manual.CreatedAt.UnixNano(), got.CreatedAt.UnixNano())

	duplicateManual, err := domain.NewManualSource(domain.SavedLinksName)
	require.NoError(t, err)
	err = repo.CreateSource(ctx, duplicateManual)
	require.Error(t, err)
	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperr.CodeConflict, appErr.Code)

	source, err := domain.NewRSSSource("Example", "https://example.test/feed")
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(ctx, source))
	duplicate, err := domain.NewRSSSource("Duplicate", source.RSSURL)
	require.NoError(t, err)
	err = repo.CreateSource(ctx, duplicate)
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperr.CodeConflict, appErr.Code)

	require.NoError(t, source.Deactivate())
	require.NoError(t, repo.UpdateSource(ctx, source))
	active, err := repo.ListSources(ctx, structFilter(false))
	require.NoError(t, err)
	require.Len(t, active, 1)
	all, err := repo.ListSources(ctx, structFilter(true))
	require.NoError(t, err)
	require.Len(t, all, 2)

	require.NoError(t, db.Close())
	db, err = sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	defer db.Close()
	repo = NewRepository(db)
	got, err = repo.GetSourceByID(ctx, source.ID)
	require.NoError(t, err)
	require.False(t, got.Active)
	require.Equal(t, source.UpdatedAt.UnixNano(), got.UpdatedAt.UnixNano())

	badID := uuid.New()
	_, err = repo.GetSourceByID(ctx, badID)
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperr.CodeNotFound, appErr.Code)
}

func structFilter(includeDeactivated bool) repository.ListSourcesFilter {
	return repository.ListSourcesFilter{IncludeDeactivated: includeDeactivated}
}

func applySourcesMigration(t *testing.T, db *sql.DB) {
	t.Helper()
	require.NoError(t, goose.SetDialect("sqlite3"))
	err := goose.UpTo(db, "../../../../migrations", 1)
	require.NoError(t, err)
}
