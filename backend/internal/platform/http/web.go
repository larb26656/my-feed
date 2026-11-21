package httpx

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// ServeWeb serves the SPA without changing API-only operation when root is empty.
func ServeWeb(r *gin.Engine, root string) {
	if root == "" {
		return
	}
	files := http.FileServer(http.Dir(root))
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead || reservedWebPath(p) {
			http.NotFound(c.Writer, c.Request)
			return
		}
		clean := path.Clean("/" + p)
		filename := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(clean, "/")))
		if info, err := os.Stat(filename); err == nil && !info.IsDir() {
			files.ServeHTTP(c.Writer, c.Request)
			return
		}
		if path.Ext(clean) != "" || strings.HasPrefix(clean, "/assets/") {
			http.NotFound(c.Writer, c.Request)
			return
		}
		http.ServeFile(c.Writer, c.Request, filepath.Join(root, "index.html"))
	})
}

func reservedWebPath(p string) bool {
	for _, prefix := range []string{"/api", "/mcp", "/healthz", "/swagger"} {
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			return true
		}
	}
	return false
}
