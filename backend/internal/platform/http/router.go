package httpx

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type requestIDKey struct{}

// RequestID returns the request identifier propagated to application services.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// NewRouter configures the HTTP contract shared by production and test routes.
func NewRouter() *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.UseRawPath = true
	r.UnescapePathValues = false
	r.HandleMethodNotAllowed = true
	_ = r.SetTrustedProxies(nil)
	r.Use(func(c *gin.Context) {
		id := c.GetHeader("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDKey{}, id))
		c.Next()
	}, gin.Recovery())
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
