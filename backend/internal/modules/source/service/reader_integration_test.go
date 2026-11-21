package service_test

import (
	"context"
	"fmt"
	"testing"

	sourcesqlite "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/sqlite"
	"github.com/larb26656/my-feed/backend/internal/modules/source/service"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
)

func TestListActiveFetchableIncludesSourcesBeyondFirstPage(t *testing.T) {
	// Arrange
	ctx := context.Background()
	pool := testkit.NewDatabase(t)
	sources := service.New(sourcesqlite.NewRepository(pool.DB))
	for i := 0; i < 12; i++ {
		_, err := sources.CreateSource(ctx, service.CreateSourceInput{Name: fmt.Sprintf("Source %d", i), Type: "rss", RSSURL: fmt.Sprintf("https://example.test/%d", i)})
		require.NoError(t, err)
	}
	// Act
	got, err := sources.ListActiveFetchable(ctx)
	// Assert
	require.NoError(t, err)
	require.Len(t, got, 12)
}
