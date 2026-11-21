package bootstrap_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/larb26656/my-feed/backend/internal/bootstrap"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
)

func TestAPIRoutesRejectMissingBearerWhenAPIKeyConfigured(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()

	// Act
	recorder := httptest.NewRecorder()
	app.Router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/entries/", nil))

	// Assert
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAPIRoutesAllowMatchingBearerWhenAPIKeyConfigured(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries/", nil)
	req.Header.Set("Authorization", "Bearer test-secret")

	// Act
	recorder := httptest.NewRecorder()
	app.Router.ServeHTTP(recorder, req)

	// Assert
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestHealthzStaysOpenWhenAPIKeyConfigured(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()

	// Act
	recorder := httptest.NewRecorder()
	app.Router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	// Assert
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestAPIRoutesOpenWhenAPIKeyEmpty(t *testing.T) {
	// Arrange
	app := newTestApp(t, "")
	defer app.Close()

	// Act
	recorder := httptest.NewRecorder()
	app.Router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/entries/", nil))

	// Assert
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestMCPRequiresBearerWhenAPIKeyConfigured(t *testing.T) {
	// Arrange
	app := newTestApp(t, "mcp-secret")
	defer app.Close()

	// Act
	denied := httptest.NewRecorder()
	app.Router.ServeHTTP(denied, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	allowedReq := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	allowedReq.Header.Set("Authorization", "Bearer mcp-secret")
	allowed := httptest.NewRecorder()
	app.Router.ServeHTTP(allowed, allowedReq)

	// Assert
	require.Equal(t, http.StatusUnauthorized, denied.Code)
	require.NotEqual(t, http.StatusUnauthorized, allowed.Code)
}

func newTestApp(t *testing.T, apiKey string) *bootstrap.App {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	seed, err := database.NewPool(context.Background(), dbPath)
	require.NoError(t, err)
	testkit.Migrate(t, seed.DB)
	require.NoError(t, seed.Close())

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	app, err := bootstrap.NewWithConfig(ctx, bootstrap.Config{
		DatabasePath: dbPath,
		HTTPAddr:     ":0",
		APIKey:       apiKey,
		Fetch: bootstrap.FetchConfig{
			RetryBackoff:     time.Hour,
			ScheduleInterval: time.Hour,
			ExecutorPoll:     time.Hour,
		},
	})
	require.NoError(t, err)
	return app
}
