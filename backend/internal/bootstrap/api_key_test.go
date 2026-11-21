package bootstrap_test

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestAPIRoutesRejectMissingBearer(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()

	// Act
	recorder := httptest.NewRecorder()
	app.Router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/entries/", nil))

	// Assert
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAPIRoutesAllowMatchingAPIKey(t *testing.T) {
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

func TestAPIRoutesAllowJWTFromLogin(t *testing.T) {
	// Arrange
	app := newTestApp(t, "")
	defer app.Close()
	access := loginAccessToken(t, app)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries/", nil)
	req.Header.Set("Authorization", "Bearer "+access)

	// Act
	recorder := httptest.NewRecorder()
	app.Router.ServeHTTP(recorder, req)

	// Assert
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestAuthLoginIsPublic(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()
	body, err := json.Marshal(map[string]string{
		"username": "admin",
		"password": "secret",
	})
	require.NoError(t, err)

	// Act
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	app.Router.ServeHTTP(recorder, req)

	// Assert
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestHealthzStaysOpenWhenAuthConfigured(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()

	// Act
	recorder := httptest.NewRecorder()
	app.Router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	// Assert
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestAPIRoutesRequireJWTWhenAPIKeyEmpty(t *testing.T) {
	// Arrange
	app := newTestApp(t, "")
	defer app.Close()

	// Act
	recorder := httptest.NewRecorder()
	app.Router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/entries/", nil))

	// Assert
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestMCPRequiresAuth(t *testing.T) {
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

	jwtReq := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	jwtReq.Header.Set("Authorization", "Bearer "+loginAccessToken(t, app))
	jwtAllowed := httptest.NewRecorder()
	app.Router.ServeHTTP(jwtAllowed, jwtReq)

	// Assert
	require.Equal(t, http.StatusUnauthorized, denied.Code)
	require.NotEqual(t, http.StatusUnauthorized, allowed.Code)
	require.NotEqual(t, http.StatusUnauthorized, jwtAllowed.Code)
}

func loginAccessToken(t *testing.T, app *bootstrap.App) string {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"username": "admin",
		"password": "secret",
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	app.Router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.True(t, envelope.Success)
	require.NotEmpty(t, envelope.Data.AccessToken)
	return envelope.Data.AccessToken
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
		Auth: bootstrap.AuthConfig{
			Username:   "admin",
			Password:   "secret",
			JWTSecret:  "jwt-test-secret",
			AccessTTL:  15 * time.Minute,
			RefreshTTL: 7 * 24 * time.Hour,
		},
		Fetch: bootstrap.FetchConfig{
			RetryBackoff:     time.Hour,
			ScheduleInterval: time.Hour,
			ExecutorPoll:     time.Hour,
		},
	})
	require.NoError(t, err)
	return app
}
