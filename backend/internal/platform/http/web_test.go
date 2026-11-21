package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/stretchr/testify/require"
)

func TestWebServesFrontendAndPreservesOtherRoutes(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "index.html"), []byte("app shell"), 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(root, "assets"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "assets", "app.js"), []byte("javascript"), 0o600))
	router := httpx.NewRouter()
	router.GET("/healthz", func(c *gin.Context) { c.String(200, "healthy") })
	httpx.ServeWeb(router, root)
	for _, tc := range []struct {
		method, path string
		status       int
		body         string
	}{
		{"GET", "/healthz", 200, "healthy"},
		{"GET", "/", 200, "app shell"},
		{"GET", "/sources/123", 200, "app shell"},
		{"GET", "/assets/app.js", 200, "javascript"},
		{"GET", "/assets/missing.js", 404, "404 page not found"},
		{"GET", "/api/missing", 404, "404 page not found"},
		{"GET", "/mcp/missing", 404, "404 page not found"},
		{"GET", "/healthz/missing", 404, "404 page not found"},
		{"POST", "/sources/123", 404, "404 page not found"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			require.Equal(t, tc.status, w.Code)
			require.Contains(t, w.Body.String(), tc.body)
		})
	}
}

func TestWebDisabledWithoutRoot(t *testing.T) {
	router := httpx.NewRouter()
	router.GET("/healthz", func(c *gin.Context) { c.Status(200) })
	httpx.ServeWeb(router, "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusNotFound, w.Code)
}
