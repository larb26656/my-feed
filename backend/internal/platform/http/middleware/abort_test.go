package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/larb26656/my-feed/backend/internal/platform/http/middleware"
	"github.com/stretchr/testify/require"
)

func TestAbortPairReRaisesErrAbortHandlerPastRecovery(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(
		middleware.AbortRecovery(gin.Recovery()),
		middleware.CatchAbortHandler(),
	)
	router.GET("/", func(c *gin.Context) {
		panic(http.ErrAbortHandler)
	})

	// Act
	var panicked any
	func() {
		defer func() { panicked = recover() }()
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}()

	// Assert
	require.Equal(t, http.ErrAbortHandler, panicked)
}

func TestAbortPairLetsRecoveryHandleOtherPanics(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(
		middleware.AbortRecovery(gin.Recovery()),
		middleware.CatchAbortHandler(),
	)
	router.GET("/", func(c *gin.Context) {
		panic("boom")
	})

	// Act
	var panicked any
	recorder := httptest.NewRecorder()
	func() {
		defer func() { panicked = recover() }()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	}()

	// Assert
	require.Nil(t, panicked)
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestAbortPairPassesThroughSuccessfulRequests(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(
		middleware.AbortRecovery(gin.Recovery()),
		middleware.CatchAbortHandler(),
	)
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	// Act
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	// Assert
	require.Equal(t, http.StatusNoContent, recorder.Code)
}
