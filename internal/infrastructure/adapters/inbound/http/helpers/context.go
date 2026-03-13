package helpers

import (
	"net/http"

	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MustGetUserID extracts the authenticated user's ID from the Gin context.
// If the ID is missing or invalid, it aborts the request with 401 and returns uuid.Nil.
// Callers should check c.IsAborted() after calling this.
//
// Usage:
//
//	func (h *UserHandler) GetMe(c *gin.Context) {
//	    userID := helpers.MustGetUserID(c)
//	    if c.IsAborted() { return }
//	    // ...
//	}
func MustGetUserID(c *gin.Context) uuid.UUID {
	value, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil
	}

	return userID
}
