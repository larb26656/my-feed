package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/platform/http/middleware"
	"github.com/stretchr/testify/require"
)

func TestRequestIDUsesIncomingHeader(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	var got string
	router := gin.New()
	router.Use(middleware.RequestID())
	router.GET("/", func(c *gin.Context) {
		got = middleware.RequestIDFromContext(c.Request.Context())
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "client-req-1")

	// Act
	router.ServeHTTP(httptest.NewRecorder(), req)

	// Assert
	require.Equal(t, "client-req-1", got)
}

func TestRequestIDGeneratesWhenHeaderMissing(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	var got string
	router := gin.New()
	router.Use(middleware.RequestID())
	router.GET("/", func(c *gin.Context) {
		got = middleware.RequestIDFromContext(c.Request.Context())
		c.Status(http.StatusNoContent)
	})

	// Act
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	// Assert
	require.NotEmpty(t, got)
	_, err := uuid.Parse(got)
	require.NoError(t, err)
}

func TestRequestIDFromContextEmptyWithoutMiddleware(t *testing.T) {
	// Act & Assert
	require.Empty(t, middleware.RequestIDFromContext(t.Context()))
}
