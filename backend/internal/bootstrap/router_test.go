package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
)

func TestProductionRouterHealthAndMiddleware(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	path := filepath.Join(t.TempDir(), "app.db")
	pool, err := database.NewPool(ctx, path)
	require.NoError(t, err)
	testkit.Migrate(t, pool.DB)
	require.NoError(t, pool.Close())
	app, err := NewWithConfig(ctx, Config{DatabasePath: path, Fetch: FetchConfig{RetryBackoff: time.Hour, ScheduleInterval: time.Hour, ExecutorPoll: time.Hour}})
	require.NoError(t, err)
	t.Cleanup(func() { cancel(); app.Close() })
	app.Router.GET("/test/context", func(c *gin.Context) { c.String(200, "%s|%s", httpx.RequestID(c.Request.Context()), c.ClientIP()) })
	app.Router.GET("/test/panic", func(c *gin.Context) { panic("test panic") })
	t.Run("health", func(t *testing.T) {
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
		require.Equal(t, 200, w.Code)
		require.Equal(t, "ok", w.Body.String())
	})
	t.Run("request id and untrusted forwarded headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test/context", nil)
		req.RemoteAddr = "192.0.2.10:1234"
		req.Header.Set("X-Request-Id", "request-123")
		req.Header.Set("X-Forwarded-For", "198.51.100.1")
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
		require.Equal(t, "request-123|192.0.2.10", w.Body.String())
		require.Empty(t, w.Header().Get("X-Request-Id"))
	})
	t.Run("generated request id", func(t *testing.T) {
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, httptest.NewRequest("GET", "/test/context", nil))
		require.NotEmpty(t, strings.SplitN(w.Body.String(), "|", 2)[0])
	})
	t.Run("recovers panics", func(t *testing.T) {
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, httptest.NewRequest("GET", "/test/panic", nil))
		require.Equal(t, http.StatusInternalServerError, w.Code)
		w = httptest.NewRecorder()
		app.Router.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
		require.Equal(t, 200, w.Code)
	})
}
