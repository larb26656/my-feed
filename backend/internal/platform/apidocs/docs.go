// Package apidocs serves the generated API contract and the bundled Scalar UI.
package apidocs

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed swagger.json assets/scalar.js
var files embed.FS

func Register(r *gin.Engine) {
	r.GET("/openapi.json", func(c *gin.Context) {
		b, _ := files.ReadFile("swagger.json")
		c.Data(http.StatusOK, "application/json; charset=utf-8", b)
	})
	r.GET("/docs/scalar.js", func(c *gin.Context) {
		b, _ := files.ReadFile("assets/scalar.js")
		c.Data(http.StatusOK, "text/javascript; charset=utf-8", b)
	})
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>My Feed API</title></head><body><div id="app"></div><script src="/docs/scalar.js"></script><script>Scalar.createApiReference("#app", {url: "/openapi.json"})</script></body></html>`))
	})
}
