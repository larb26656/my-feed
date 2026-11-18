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

func newRouterTestApp(t *testing.T) *gin.Engine {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	path := filepath.Join(t.TempDir(), "app.db")
	pool, err := database.NewPool(ctx, path)
	require.NoError(t, err)
	testkit.Migrate(t, pool.DB)
	require.NoError(t, pool.Close())
	app, err := NewWithConfig(ctx, Config{DatabasePath: path, Fetch: FetchConfig{RetryBackoff: time.Hour, ScheduleInterval: time.Hour, ExecutorPoll: time.Hour}})
	require.NoError(t, err)
	t.Cleanup(func() { cancel(); app.Close() })
	return app.Router
}

func TestProductionRouterHealthAndMiddleware(t *testing.T) {
	t.Run("health returns ok", func(t *testing.T) {
		// Arrange
		router := newRouterTestApp(t)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/healthz", nil)

		// Act
		router.ServeHTTP(w, req)

		// Assert
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "ok", w.Body.String())
	})
	t.Run("request id is propagated without a response header", func(t *testing.T) {
		// Arrange
		router := newRouterTestApp(t)
		router.GET("/test/context", func(c *gin.Context) { c.String(200, "%s", httpx.RequestID(c.Request.Context())) })
		req := httptest.NewRequest("GET", "/test/context", nil)
		req.Header.Set("X-Request-Id", "request-123")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		require.Equal(t, "request-123", w.Body.String())
		require.Empty(t, w.Header().Get("X-Request-Id"))
	})
	t.Run("forwarded client ip is untrusted", func(t *testing.T) {
		// Arrange
		router := newRouterTestApp(t)
		router.GET("/test/ip", func(c *gin.Context) { c.String(200, "%s", c.ClientIP()) })
		req := httptest.NewRequest("GET", "/test/ip", nil)
		req.RemoteAddr = "192.0.2.10:1234"
		req.Header.Set("X-Forwarded-For", "198.51.100.1")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		require.Equal(t, "192.0.2.10", w.Body.String())
	})
	t.Run("missing request id is generated", func(t *testing.T) {
		// Arrange
		router := newRouterTestApp(t)
		router.GET("/test/context", func(c *gin.Context) { c.String(200, "%s", httpx.RequestID(c.Request.Context())) })
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test/context", nil)

		// Act
		router.ServeHTTP(w, req)

		// Assert
		require.NotEmpty(t, strings.TrimSpace(w.Body.String()))
	})
	t.Run("ordinary panic returns internal server error", func(t *testing.T) {
		// Arrange
		router := newRouterTestApp(t)
		router.GET("/test/panic", func(c *gin.Context) { panic("test panic") })
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test/panic", nil)

		// Act
		router.ServeHTTP(w, req)

		// Assert
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("abort handler panic reaches the http server", func(t *testing.T) {
		// Arrange
		router := newRouterTestApp(t)
		router.GET("/test/abort", func(c *gin.Context) { panic(http.ErrAbortHandler) })
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test/abort", nil)

		// Act
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			router.ServeHTTP(w, req)
		}()

		// Assert
		require.Equal(t, http.ErrAbortHandler, recovered)
		require.Empty(t, w.Body.String())
	})
}
