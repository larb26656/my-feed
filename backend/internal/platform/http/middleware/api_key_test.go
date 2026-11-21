package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/larb26656/my-feed/backend/internal/platform/http/middleware"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyAllowsRequestsWhenSecretEmpty(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorMiddleware(), middleware.APIKey(""))
	router.GET("/api/v1/entries", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	// Act
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil))

	// Assert
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestAPIKeyRejectsMissingBearerWhenSecretSet(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorMiddleware(), middleware.APIKey("secret"))
	router.GET("/api/v1/entries", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	// Act
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil))

	// Assert
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.JSONEq(t, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"missing or invalid Authorization bearer token"}}`, recorder.Body.String())
}

func TestAPIKeyRejectsWrongBearerWhenSecretSet(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorMiddleware(), middleware.APIKey("secret"))
	router.GET("/api/v1/entries", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
	req.Header.Set("Authorization", "Bearer wrong")

	// Act
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	// Assert
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.JSONEq(t, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"missing or invalid Authorization bearer token"}}`, recorder.Body.String())
}

func TestAPIKeyAllowsMatchingBearerWhenSecretSet(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorMiddleware(), middleware.APIKey("secret"))
	router.GET("/api/v1/entries", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
	req.Header.Set("Authorization", "Bearer secret")

	// Act
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	// Assert
	require.Equal(t, http.StatusNoContent, recorder.Code)
}
