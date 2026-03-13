package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware logs each HTTP request with structured fields.
//
// Output example (JSON format):
//
//	{"time":"2026-03-09T10:00:00Z","level":"INFO","msg":"request completed",
//	 "request_id":"abc-123","method":"POST","path":"/api/v1/users",
//	 "status":201,"duration_ms":23.45,"client_ip":"192.168.1.1"}
//
// Log level is chosen based on status code:
//   - 5xx → ERROR
//   - 4xx → WARN
//   - 2xx/3xx → INFO
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
		status := c.Writer.Status()
		duration := time.Since(start)
		requestID := GetRequestIDFromContext(c)

		// Choose log level based on status code
		attrs := []slog.Attr{
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Float64("duration_ms", float64(duration.Microseconds())/1000.0),
			slog.String("client_ip", c.ClientIP()),
		}

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		slog.LogAttrs(c.Request.Context(), level, "request completed", attrs...)
	}
}
