package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type errorResponse struct {
	Success bool      `json:"success"`
	Error   errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail,omitempty"`
}

func fail(code, message string, details ...any) errorResponse {
	var detail any
	if len(details) > 0 {
		detail = details[0]
	}
	return errorResponse{
		Success: false,
		Error: errorBody{
			Code:    code,
			Message: message,
			Detail:  detail,
		},
	}
}

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
			c.AbortWithStatusJSON(http.StatusInternalServerError, fail("INTERNAL_SERVER_ERROR", "Internal server error"))
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
			c.AbortWithStatusJSON(status, fail("INTERNAL_SERVER_ERROR", "Internal server error"))
			return
		}

		c.AbortWithStatusJSON(status, fail(appErr.Code, appErr.Message, appErr.Detail))
	}
}
