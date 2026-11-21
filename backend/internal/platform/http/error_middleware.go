package httpx

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

// ErrorMiddleware writes the last error reported with c.Error after a handler returns.
func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		err := c.Errors.Last().Err
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, Fail("INTERNAL_SERVER_ERROR", "Internal server error"))
			return
		}

		status := http.StatusInternalServerError
		switch appErr.Kind {
		case apperr.KindInvalid:
			status = http.StatusBadRequest
		case apperr.KindNotFound:
			status = http.StatusNotFound
		case apperr.KindConflict:
			status = http.StatusConflict
		case apperr.KindUnauthorized:
			status = http.StatusUnauthorized
		case apperr.KindForbidden:
			status = http.StatusForbidden
		case apperr.KindRateLimited:
			status = http.StatusTooManyRequests
		case apperr.KindUnavailable:
			status = http.StatusServiceUnavailable
		}

		if status == http.StatusInternalServerError {
			c.AbortWithStatusJSON(status, Fail("INTERNAL_SERVER_ERROR", "Internal server error"))
			return
		}

		c.AbortWithStatusJSON(status, Fail(appErr.Code, appErr.Message, appErr.Detail))
	}
}
