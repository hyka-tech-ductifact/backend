package helpers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DomainError represents a business error that can be mapped to an HTTP status code.
// Domain and application errors are registered in the errorMap below.
type DomainError struct {
	StatusCode int
	Message    string
}

// errorMap is the single source of truth for mapping domain errors to HTTP status codes.
// When you add a new domain error, register it here.
var errorMap = map[error]DomainError{}

// RegisterDomainError maps a domain/application error to an HTTP status code.
// Call this during initialization (in router.go or main.go) to register all known errors.
//
// Example:
//
//	helpers.RegisterDomainError(services.ErrUserNotFound, http.StatusNotFound, "user not found")
//	helpers.RegisterDomainError(services.ErrEmailAlreadyInUse, http.StatusConflict, "email already in use")
func RegisterDomainError(err error, statusCode int, message string) {
	errorMap[err] = DomainError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// HandleError maps a domain/application error to the appropriate HTTP response.
// If the error is registered in the errorMap, it returns the mapped status code and message.
// If the error is unknown, it returns 500 Internal Server Error.
//
// Usage in handlers:
//
//	user, err := h.userService.GetUserByID(ctx, id)
//	if err != nil {
//	    helpers.HandleError(c, err)
//	    return
//	}
func HandleError(c *gin.Context, err error) {
	for domainErr, httpErr := range errorMap {
		if errors.Is(err, domainErr) {
			c.JSON(httpErr.StatusCode, gin.H{"error": httpErr.Message})
			return
		}
	}

	// Unknown error → 500 (don't expose internal details)
	// Log the real error so we can debug it. The client only sees a generic message.
	log.Printf("[ERROR] unhandled error: %v (path: %s %s)", err, c.Request.Method, c.Request.URL.Path)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
