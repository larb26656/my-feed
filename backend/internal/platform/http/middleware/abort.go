package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type abortHandlerKey struct{}

// AbortRecovery runs recovery around the remaining chain and re-raises
// http.ErrAbortHandler after recovery returns so net/http can handle the abort.
// Pair with CatchAbortHandler immediately inside it.
func AbortRecovery(recovery gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Carry abort panics past Gin's recovery without logging or swallowing them.
		aborted := false
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), abortHandlerKey{}, &aborted))
		recovery(c)
		if aborted {
			panic(http.ErrAbortHandler)
		}
	}
}

// CatchAbortHandler converts http.ErrAbortHandler into an aborted request and
// signals AbortRecovery to re-raise it. Other panics are re-panicked for recovery.
func CatchAbortHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		aborted := c.Request.Context().Value(abortHandlerKey{}).(*bool)
		defer func() {
			if err := recover(); err != nil {
				if err != http.ErrAbortHandler {
					panic(err)
				}
				*aborted = true
				c.Abort()
			}
		}()
		c.Next()
	}
}
