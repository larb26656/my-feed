package httpx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/stretchr/testify/require"
)

func TestErrorMiddlewareReturnsValidationDetails(t *testing.T) {
	// Arrange
	recorder := httptest.NewRecorder()
	err := fmt.Errorf("entry: %w", apperr.InvalidInput.New("title is required"))

	// Act
	router := gin.New()
	router.Use(httpx.ErrorMiddleware())
	router.GET("/", func(c *gin.Context) { _ = c.Error(err) })
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	// Assert
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.JSONEq(t, `{"success":false,"error":{"message":"title is required","code":"VALIDATION"}}`, recorder.Body.String())
}

func TestErrorMiddlewareReturnsEntryNotFound(t *testing.T) {
	// Arrange
	recorder := httptest.NewRecorder()
	err := fmt.Errorf("lookup: %w", apperr.ResourceNotFound.New("entry not found"))

	// Act
	router := gin.New()
	router.Use(httpx.ErrorMiddleware())
	router.GET("/", func(c *gin.Context) { _ = c.Error(err) })
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	// Assert
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.JSONEq(t, `{"success":false,"error":{"message":"entry not found","code":"NOT_FOUND"}}`, recorder.Body.String())
}
