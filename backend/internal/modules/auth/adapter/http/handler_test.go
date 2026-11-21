package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authhttp "github.com/larb26656/my-feed/backend/internal/modules/auth/adapter/http"
	authsqlite "github.com/larb26656/my-feed/backend/internal/modules/auth/adapter/sqlite"
	"github.com/larb26656/my-feed/backend/internal/modules/auth/service"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
)

func TestLoginRefreshLogoutHTTP(t *testing.T) {
	// Arrange
	router := newAuthRouter(t)

	// Act: login
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginRec := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(loginRec, loginReq)

	// Assert login
	require.Equal(t, http.StatusOK, loginRec.Code)
	tokens := decodeTokens(t, loginRec.Body.Bytes())
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
	require.Equal(t, "Bearer", tokens.TokenType)
	require.Equal(t, int64(900), tokens.ExpiresIn)

	// Act: refresh
	refreshBody, _ := json.Marshal(map[string]string{"refresh_token": tokens.RefreshToken})
	refreshRec := httptest.NewRecorder()
	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
	refreshReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(refreshRec, refreshReq)

	// Assert refresh
	require.Equal(t, http.StatusOK, refreshRec.Code)
	rotated := decodeTokens(t, refreshRec.Body.Bytes())
	require.NotEqual(t, tokens.RefreshToken, rotated.RefreshToken)

	// Act: logout
	logoutBody, _ := json.Marshal(map[string]string{"refresh_token": rotated.RefreshToken})
	logoutRec := httptest.NewRecorder()
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(logoutBody))
	logoutReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(logoutRec, logoutReq)

	// Assert logout
	require.Equal(t, http.StatusOK, logoutRec.Code)

	// Act: refresh after logout
	againRec := httptest.NewRecorder()
	againReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(logoutBody))
	againReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(againRec, againReq)

	// Assert
	require.Equal(t, http.StatusUnauthorized, againRec.Code)
}

func newAuthRouter(t *testing.T) http.Handler {
	t.Helper()
	pool := testkit.NewDatabase(t)
	svc := service.New(authsqlite.NewRepository(pool.DB), pool, service.Config{
		Username:   "admin",
		Password:   "secret",
		JWTSecret:  "jwt-test-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	router := httpx.NewRouter()
	authhttp.NewHandler(svc).RegisterRoutes(router.Group("/api/v1/auth"))
	return router
}

type tokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func decodeTokens(t *testing.T, body []byte) tokenData {
	t.Helper()
	var envelope struct {
		Success bool      `json:"success"`
		Data    tokenData `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}
