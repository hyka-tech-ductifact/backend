package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// RecoveryMiddleware catches panics in handlers and returns a clean 500 error
// instead of crashing the server.
//
// It logs the panic message and stack trace for debugging,
// and includes the request ID if available for traceability.
//
// This middleware should be registered early in the chain so that it can
// catch panics from any downstream middleware or handler.
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				requestID := GetRequestIDFromContext(c)

				// Log the panic with stack trace for debugging
				log.Printf("[req:%s] PANIC recovered: %v\n%s",
					requestID,
					r,
					debug.Stack(),
				)

				// Return a clean 500 to the client (don't expose internal details)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":      "internal server error",
					"request_id": requestID,
				})
			}
		}()

		c.Next()
	}
}
