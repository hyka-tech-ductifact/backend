package middleware

import (
	"net/http"
	"strings"

	"ductifact/internal/application/ports"

	"github.com/gin-gonic/gin"
)

// UserIDKey is the key used to store/retrieve the authenticated user's ID in Gin's context.
const UserIDKey contextKey = "userID"

// AuthMiddleware creates a Gin middleware that validates JWT tokens.
// It extracts the token from the Authorization header, validates it,
// and puts the userID in the request context for downstream handlers.
//
// This is an inbound adapter — it does NOT implement any interface.
// It consumes ports.TokenProvider (an outbound port) to validate tokens,
// and translates HTTP authentication concerns into application-level context.
func AuthMiddleware(tokenProvider ports.TokenProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Step 1: Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header required",
			})
			return
		}

		// Step 2: Extract the token (expected format: "Bearer <token>")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header must be: Bearer <token>",
			})
			return
		}

		tokenString := parts[1]

		// Step 3: Validate the token via the outbound port
		claims, err := tokenProvider.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			return
		}

		// Step 4: Put the userID in Gin's context (available to all downstream handlers)
		c.Set(string(UserIDKey), claims.UserID)

		// Step 5: Continue to the next handler in the chain
		c.Next()
	}
}
