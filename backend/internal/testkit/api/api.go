// Package api wires Entry and Source HTTP tests without starting fetch workers.
package api

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	entryhttp "github.com/larb26656/my-feed/backend/internal/entry/adapter/http"
	entrysqlite "github.com/larb26656/my-feed/backend/internal/entry/adapter/sqlite"
	entrysvc "github.com/larb26656/my-feed/backend/internal/entry/service"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	sourcehttp "github.com/larb26656/my-feed/backend/internal/source/adapter/http"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/source/adapter/sqlite"
	sourcedomain "github.com/larb26656/my-feed/backend/internal/source/domain"
	sourcerepo "github.com/larb26656/my-feed/backend/internal/source/repository"
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
	entries := entrysvc.New(entrysqlite.NewRepository(pool.DB), sourceLookup{sourceRepo})
	router := httpx.NewRouter()
	sourcehttp.NewHandler(sources).RegisterRoutes(router.Group("/api/v1/sources"))
	entryhttp.NewHandler(entries).RegisterRoutes(router.Group("/api/v1/entries"))
	return router
}

type sourceLookup struct{ repo sourcerepo.Repository }

func ref(source *sourcedomain.Source, err error) (*entrysvc.SourceRef, error) {
	if err != nil {
		return nil, err
	}
	return &entrysvc.SourceRef{ID: source.ID, Type: entrysvc.SourceType(source.Type), Active: source.Active}, nil
}
func (s sourceLookup) Get(ctx context.Context, id uuid.UUID) (*entrysvc.SourceRef, error) {
	return ref(s.repo.GetSourceByID(ctx, id))
}
func (s sourceLookup) GetManual(ctx context.Context) (*entrysvc.SourceRef, error) {
	return ref(s.repo.GetManualSource(ctx))
}
