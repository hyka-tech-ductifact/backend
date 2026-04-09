package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// localhostWildcard is the special CORS_ORIGINS value that enables
// http://localhost on any port (for frontend development).
const localhostWildcard = "http://localhost"

// CORSMiddleware returns a gin middleware that enforces CORS based on the
// configured allowed origins.
//
// Origin matching rules:
//   - Exact match: any origin in allowedOrigins is permitted as-is.
//   - Localhost wildcard: if "http://localhost" is present, any
//     http://localhost:<port> origin is permitted. This lets frontend
//     developers work against a shared backend without listing every port.
//     Omit it in production to block localhost entirely.
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	matcher := buildOriginMatcher(allowedOrigins)

	return cors.New(cors.Config{
		AllowOriginFunc:  matcher,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

// buildOriginMatcher parses the allowed origins list and returns a function
// that decides whether a given request origin should be permitted.
func buildOriginMatcher(origins []string) func(string) bool {
	exactSet := make(map[string]bool, len(origins))
	allowLocalhost := false

	for _, o := range origins {
		if o == localhostWildcard {
			allowLocalhost = true
		} else {
			exactSet[o] = true
		}
	}

	return func(origin string) bool {
		if exactSet[origin] {
			return true
		}
		return allowLocalhost && strings.HasPrefix(origin, "http://localhost:")
	}
}
