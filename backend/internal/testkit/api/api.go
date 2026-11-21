// Package api wires Entry and Source HTTP tests without starting fetch workers.
package api

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	entryhttp "github.com/larb26656/my-feed/backend/internal/entry/adapter/http"
	entrysqlite "github.com/larb26656/my-feed/backend/internal/entry/adapter/sqlite"
	entrysvc "github.com/larb26656/my-feed/backend/internal/entry/service"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	sourcehttp "github.com/larb26656/my-feed/backend/internal/source/adapter/http"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/source/adapter/sqlite"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/source/service"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
)

// New mounts the production module routes and services on Gin. Bootstrap tests
// separately cover fetch hooks and worker wiring.
func New(t testing.TB) *gin.Engine {
	t.Helper()
	pool := testkit.NewDatabase(t)
	sourceRepo := sourcesqlite.NewRepository(pool.DB)
	sources := sourcesvc.New(sourceRepo)
	_, err := sources.EnsureSavedLinks(context.Background())
	require.NoError(t, err)
	entries := entrysvc.New(entrysqlite.NewRepository(pool.DB), sources)
	router := httpx.NewRouter()
	sourcehttp.NewHandler(sources, sources).RegisterRoutes(router.Group("/api/v1/sources"))
	entryhttp.NewHandler(entries).RegisterRoutes(router.Group("/api/v1/entries"))
	return router
}
