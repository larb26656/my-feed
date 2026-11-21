package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/larb26656/my-feed/backend/internal/platform/http/middleware"
	"github.com/stretchr/testify/require"
)

func TestAuthenticateAcceptsValidJWT(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	secret := "jwt-secret"
	router := gin.New()
	router.Use(middleware.ErrorMiddleware(), middleware.Authenticate(secret, ""))
	router.GET("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	token := signAccess(t, secret, "admin", time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Act
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	// Assert
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestAuthenticateFallsBackToAPIKey(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorMiddleware(), middleware.Authenticate("jwt-secret", "api-key"))
	router.GET("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer api-key")

	// Act
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	// Assert
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestAuthenticateRejectsWhenNeitherJWTNorAPIKey(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorMiddleware(), middleware.Authenticate("jwt-secret", "api-key"))
	router.GET("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer nope")

	// Act
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	// Assert
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAuthenticateRequiresBearerEvenWithoutAPIKey(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorMiddleware(), middleware.Authenticate("jwt-secret", ""))
	router.GET("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	// Act
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/x", nil))

	// Assert
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func signAccess(t *testing.T, secret, sub string, ttl time.Duration) string {
	t.Helper()
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   sub,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	})
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}
