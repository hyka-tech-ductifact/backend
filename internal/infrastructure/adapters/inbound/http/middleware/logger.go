package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware logs each HTTP request with method, path, status code,
// duration, and client IP. It also includes the request ID if available.
//
// Example output:
//
//	[req:abc-123] POST   /api/v1/users  201  23.45ms  192.168.1.1
//	[req:def-456] GET    /api/v1/health 200   1.23ms  192.168.1.1
//
// This middleware must be registered AFTER RequestIDMiddleware
// so that the request ID is available in the context.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- BEFORE handler ---
		start := time.Now()
		path := c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			path = path + "?" + c.Request.URL.RawQuery
		}

		// Execute the handler (and all remaining middlewares)
		c.Next()

		// --- AFTER handler ---
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		requestID := GetRequestIDFromContext(c)

		log.Printf("[req:%s] %-6s %s %d %v %s",
			requestID,
			method,
			path,
			statusCode,
			duration,
			clientIP,
		)
	}
}
