// Package api wires Entry and Source HTTP tests without starting fetch workers.
package api

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	rss "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/rss"
	sourceyoutube "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/youtube"

	"github.com/gin-gonic/gin"
	entryhttp "github.com/larb26656/my-feed/backend/internal/modules/entry/adapter/http"
	entrysqlite "github.com/larb26656/my-feed/backend/internal/modules/entry/adapter/sqlite"
	entrysvc "github.com/larb26656/my-feed/backend/internal/modules/entry/service"
	filestorage "github.com/larb26656/my-feed/backend/internal/modules/file/service"
	sourcehttp "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/http"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/sqlite"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/modules/source/service"
	typehttp "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/http"
	typeapi "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"
	typesvc "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/service"
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
	return NewWithDefinitions(t, uploads)
}

// NewWithDB mounts production routes and returns the database they use so a
// test can place explicit timestamps before listing Entries.
func NewWithDB(t testing.TB) (*gin.Engine, *sql.DB) {
	t.Helper()
	return newEngine(t, t.TempDir())
}

func NewWithDefinitions(t testing.TB, uploads string, extra ...typeapi.Definition) *gin.Engine {
	t.Helper()
	engine, _ := newEngine(t, uploads, extra...)
	return engine
}

func newEngine(t testing.TB, uploads string, extra ...typeapi.Definition) (*gin.Engine, *sql.DB) {
	t.Helper()
	pool := testkit.NewDatabase(t)
	sourceRepo := sourcesqlite.NewRepository(pool.DB)
	sources := sourcesvc.New(sourceRepo)
	sources.SetFileStorage(filestorage.New(uploads))
	_, err := sources.EnsureSavedLinks(context.Background())
	require.NoError(t, err)
	entries := entrysvc.New(entrysqlite.NewRepository(pool.DB), sources)
	router := httpx.NewRouter()
	handler := sourcehttp.NewHandler(sources, sources)
	handler.RegisterRoutes(router.Group("/api/v1/sources"))
	typeRegistry, err := typesvc.New(append([]typeapi.Definition{rss.Definition(http.DefaultClient), sourceyoutube.Definition(http.DefaultClient)}, extra...)...)
	require.NoError(t, err)
	sources.SetSourceTypes(typeRegistry)
	sources.SetObtainer(typeRegistry)
	typehttp.RegisterRoutes(router.Group("/api/v1"), typeRegistry)
	entryhttp.NewHandler(entries).RegisterRoutes(router.Group("/api/v1/entries"))
	return router, pool.DB
}
