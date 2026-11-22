// Package api wires Entry and Source HTTP tests without starting fetch workers.
package api

import (
	"context"
	"net/http"
	"testing"

	previewhttp "github.com/larb26656/my-feed/backend/internal/modules/sourcepreview/adapter/http"
	sourcepreview "github.com/larb26656/my-feed/backend/internal/modules/sourcepreview/service"

	"github.com/larb26656/my-feed/backend/internal/modules/fetch/adapter/obtain"
	rss "github.com/larb26656/my-feed/backend/internal/modules/fetch/adapter/obtain/rss"
	youtube "github.com/larb26656/my-feed/backend/internal/modules/fetch/adapter/obtain/youtube"
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"

	"github.com/gin-gonic/gin"
	entryhttp "github.com/larb26656/my-feed/backend/internal/modules/entry/adapter/http"
	entrysqlite "github.com/larb26656/my-feed/backend/internal/modules/entry/adapter/sqlite"
	entrysvc "github.com/larb26656/my-feed/backend/internal/modules/entry/service"
	filestorage "github.com/larb26656/my-feed/backend/internal/modules/file/service"
	sourcehttp "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/http"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/sqlite"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/modules/source/service"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
)

// New mounts the production module routes and services on Gin. Bootstrap tests
// separately cover fetch hooks and worker wiring.
func New(t testing.TB) *gin.Engine {
	t.Helper()
	return NewWithUploads(t, t.TempDir())
}

func NewWithUploads(t testing.TB, uploads string) *gin.Engine {
	t.Helper()
	pool := testkit.NewDatabase(t)
	sourceRepo := sourcesqlite.NewRepository(pool.DB)
	sources := sourcesvc.New(sourceRepo)
	sources.SetFileStorage(filestorage.New(uploads))
	_, err := sources.EnsureSavedLinks(context.Background())
	require.NoError(t, err)
	entries := entrysvc.New(entrysqlite.NewRepository(pool.DB), sources)
	router := httpx.NewRouter()
	registry := obtain.NewRegistry()
	registry.Register(sourceapi.SourceTypeRSS, rss.New(http.DefaultClient))
	registry.Register(sourceapi.SourceTypeYouTube, youtube.New(http.DefaultClient))
	handler := sourcehttp.NewHandler(sources, sources)
	previewhttp.NewHandler(sourcepreview.New(sources, registry)).RegisterRoutes(router.Group("/api/v1/sources"))
	handler.RegisterRoutes(router.Group("/api/v1/sources"))
	entryhttp.NewHandler(entries).RegisterRoutes(router.Group("/api/v1/entries"))
	return router
}
