package mcp_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	entrymcp "github.com/larb26656/my-feed/backend/internal/modules/entry/adapter/mcp"
	entrysqlite "github.com/larb26656/my-feed/backend/internal/modules/entry/adapter/sqlite"
	entrysvc "github.com/larb26656/my-feed/backend/internal/modules/entry/service"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/sqlite"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/modules/source/service"
	"github.com/larb26656/my-feed/backend/internal/testkit"
)

func TestSaveLinkToolRecordsEntryWithoutMarkingSaved(t *testing.T) {
	// Arrange
	ctx := context.Background()
	entries := newEntryService(t)
	httpServer := httptest.NewServer(entrymcp.NewHandler(entries))
	t.Cleanup(httpServer.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	// Act
	created, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "save_link",
		Arguments: map[string]any{
			"title": "Example Article",
			"link":  "https://example.test/article",
		},
	})
	require.NoError(t, err)
	require.False(t, created.IsError)

	// Assert
	out := decodeToolOutput(t, created)
	require.True(t, out.Created)
	require.Equal(t, "Example Article", out.Title)
	require.Equal(t, "https://example.test/article", out.Link)
	id, err := uuid.Parse(out.ID)
	require.NoError(t, err)

	stored, err := entries.GetEntry(ctx, id)
	require.NoError(t, err)
	require.Nil(t, stored.SavedAt)

	// Act: record the same link again
	refreshed, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "save_link",
		Arguments: map[string]any{
			"title": "Example Article Updated",
			"link":  "https://example.test/article",
		},
	})
	require.NoError(t, err)
	require.False(t, refreshed.IsError)

	// Assert
	again := decodeToolOutput(t, refreshed)
	require.False(t, again.Created)
	require.Equal(t, out.ID, again.ID)
	require.Equal(t, "Example Article Updated", again.Title)
}

func newEntryService(t *testing.T) *entrysvc.Service {
	t.Helper()
	pool := testkit.NewDatabase(t)
	sources := sourcesvc.New(sourcesqlite.NewRepository(pool.DB))
	_, err := sources.EnsureSavedLinks(context.Background())
	require.NoError(t, err)
	return entrysvc.New(entrysqlite.NewRepository(pool.DB), sources)
}

type toolOut struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Link    string `json:"link"`
	Created bool   `json:"created"`
}

func decodeToolOutput(t *testing.T, result *mcp.CallToolResult) toolOut {
	t.Helper()
	require.NotNil(t, result.StructuredContent)
	raw, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	var out toolOut
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}
