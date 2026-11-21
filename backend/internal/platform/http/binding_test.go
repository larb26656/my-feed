package httpx_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/larb26656/my-feed/backend/internal/platform/http/middleware"
	"github.com/stretchr/testify/require"
)

type bindSample struct {
	Title string `json:"title" binding:"required"`
}

func TestBindJSONPopulatesDestinationOnValidBody(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"title":"Hello"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	var dst bindSample

	// Act
	ok := httpx.BindJSON(c, &dst)

	// Assert
	require.True(t, ok)
	require.Equal(t, "Hello", dst.Title)
	require.Empty(t, c.Errors)
}

func TestBindJSONRejectsMalformedJSON(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	// Act
	router := gin.New()
	router.Use(middleware.ErrorMiddleware())
	router.POST("/", func(c *gin.Context) {
		var dst bindSample
		if !httpx.BindJSON(c, &dst) {
			return
		}
		c.Status(http.StatusOK)
	})
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{`)))

	// Assert
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.JSONEq(t, `{"success":false,"error":{"message":"Invalid JSON body","code":"VALIDATION"}}`, recorder.Body.String())
}

func TestBindJSONRejectsMissingRequiredField(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	// Act
	router := gin.New()
	router.Use(middleware.ErrorMiddleware())
	router.POST("/", func(c *gin.Context) {
		var dst bindSample
		if !httpx.BindJSON(c, &dst) {
			return
		}
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	// Assert
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.JSONEq(t,
		`{"success":false,"error":{"message":"Validation failed","code":"VALIDATION","detail":{"fields":[{"field":"Title","rule":"required"}]}}}`,
		recorder.Body.String(),
	)
}
