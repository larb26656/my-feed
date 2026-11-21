package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

// APIKey requires Authorization: Bearer <secret> when secret is non-empty.
// An empty secret disables authentication.
func APIKey(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			c.Next()
			return
		}
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
			_ = c.Error(apperr.Unauthorized.New("missing or invalid Authorization bearer token"))
			c.Abort()
			return
		}
		c.Next()
	}
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}
