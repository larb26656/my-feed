package service_test

import (
	"context"
	"fmt"
	"testing"

	sourcesqlite "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/sqlite"
	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/source/service"
	rss "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/rss"
	typesvc "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/service"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
)

func TestListActiveFetchableIncludesSourcesBeyondFirstPage(t *testing.T) {
	// Arrange
	ctx := context.Background()
	pool := testkit.NewDatabase(t)
	sources := service.New(sourcesqlite.NewRepository(pool.DB))
	registry, err := typesvc.New(rss.Definition(nil))
	require.NoError(t, err)
	sources.SetSourceTypes(registry)
	for i := 0; i < 12; i++ {
		_, err := sources.CreateSource(ctx, service.CreateSourceInput{Name: fmt.Sprintf("Source %d", i), Type: "rss", Config: map[string]any{"url": fmt.Sprintf("https://example.test/%d", i)}})
		require.NoError(t, err)
	}
	// Act
	got, err := sources.ListActiveFetchable(ctx)
	// Assert
	require.NoError(t, err)
	require.Len(t, got, 12)
}

func TestRenameLegacySourceWithEmptyStoredURL(t *testing.T) {
	ctx := context.Background()
	pool := testkit.NewDatabase(t)
	repo := sourcesqlite.NewRepository(pool.DB)
	legacy, err := domain.NewRSSSource("Old", "https://example.test/feed")
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(ctx, legacy))
	_, err = pool.Exec(`UPDATE sources SET config = '{"url":""}', key = '' WHERE id = ?`, legacy.ID.String())
	require.NoError(t, err)
	updated, err := service.New(repo).UpdateSource(ctx, legacy.ID, service.UpdateSourceInput{Name: "Renamed"})
	require.NoError(t, err)
	require.Equal(t, "Renamed", updated.Name)
	require.Equal(t, map[string]any{"url": ""}, updated.Config)
}
