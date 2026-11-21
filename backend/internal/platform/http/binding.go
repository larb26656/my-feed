package httpx

import "github.com/gin-gonic/gin"

// BindJSON binds and validates a request, passing failures to ErrorMiddleware.
func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		_ = c.Error(InvalidRequest(err))
		return false
	}

	return true
}
