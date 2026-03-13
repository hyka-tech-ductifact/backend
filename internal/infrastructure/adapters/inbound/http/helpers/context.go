package helpers

import (
	"errors"

	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrUserIDNotInContext is returned when the middleware has not set the userID
// in the context — typically because the handler is not behind AuthMiddleware.
var ErrUserIDNotInContext = errors.New("user ID not found in context")

// GetUserIDFromContext extracts the authenticated user's ID from the Gin context.
// This should only be called in handlers that are behind AuthMiddleware.
//
// Usage:
//
//	func (h *UserHandler) GetMe(c *gin.Context) {
//	    userID, err := helpers.GetUserIDFromContext(c)
//	    if err != nil { ... }
//	    // userID is guaranteed to be the authenticated user
//	}
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	value, exists := c.Get(string(middleware.UserIDKey))
	if !exists {
		return uuid.Nil, ErrUserIDNotInContext
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrUserIDNotInContext
	}

	return userID, nil
}
