package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// contextKey is a private type to avoid key collisions in Gin's context.
type contextKey string

// RequestIDKey is the key used to store/retrieve the request ID in Gin's context.
const RequestIDKey contextKey = "requestID"

// RequestIDHeader is the HTTP header name used to propagate the request ID.
const RequestIDHeader = "X-Request-ID"

// RequestIDMiddleware assigns a unique UUID to each incoming request.
//
// The ID is:
//   - Stored in Gin's context (available to all downstream handlers/middlewares)
//   - Set as a response header (X-Request-ID) so the client can reference it
//
// If the client already sends an X-Request-ID header, we respect it.
// This is useful when requests pass through multiple services (microservices).
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the client already sent a request ID
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Store in context for downstream use (logger, handlers, etc.)
		c.Set(string(RequestIDKey), requestID)

		// Set as response header so the client can see it
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestIDFromContext extracts the request ID from the Gin context.
// Returns an empty string if not found (middleware not applied).
func GetRequestIDFromContext(c *gin.Context) string {
	value, exists := c.Get(string(RequestIDKey))
	if !exists {
		return ""
	}

	requestID, ok := value.(string)
	if !ok {
		return ""
	}

	return requestID
}
