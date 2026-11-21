package httpx

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/larb26656/my-feed/backend/internal/platform/http/middleware"
)

// NewRouter configures the HTTP contract shared by production and test routes.
func NewRouter() *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.UseRawPath = true
	r.UnescapePathValues = false
	r.HandleMethodNotAllowed = true
	_ = r.SetTrustedProxies(nil)
	recovery := gin.Recovery()
	r.Use(
		middleware.RequestID(),
		middleware.AbortRecovery(recovery),
		middleware.CatchAbortHandler(),
		middleware.ErrorMiddleware(),
	)
	r.NoRoute(func(c *gin.Context) { http.NotFound(c.Writer, c.Request) })
	r.NoMethod(func(c *gin.Context) {
		methods := strings.Split(c.Writer.Header().Get("Allow"), ", ")
		c.Writer.Header().Del("Allow")
		for _, method := range methods {
			if method != "" {
				c.Writer.Header().Add("Allow", method)
			}
		}
		c.Status(http.StatusMethodNotAllowed)
		c.Writer.WriteHeaderNow()
	})
	return r
}
