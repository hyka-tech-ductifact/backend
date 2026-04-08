package middleware

import (
	"log/slog"
	"net/http"

	"ductifact/internal/application/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// IPRateLimitMiddleware creates a Gin middleware that limits requests per client IP.
// Applied globally (before authentication), it prevents any single IP from
// overwhelming the API. When the limit is exceeded it returns 429 Too Many Requests.
//
// This is an inbound adapter — it consumes ports.RateLimiter (outbound port)
// and translates HTTP rate-limiting concerns into infrastructure-level decisions.
func IPRateLimitMiddleware(limiter ports.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.Allow("ip:" + ip) {
			slog.Warn("rate limit exceeded",
				"type", "ip",
				"ip", ip,
				"path", c.Request.URL.Path,
			)

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
			})
			return
		}

		c.Next()
	}
}

// UserRateLimitMiddleware creates a Gin middleware that limits requests per
// authenticated user. Applied AFTER the auth middleware, it uses the user ID
// from the JWT token as the rate-limiting key. This prevents any single user
// from consuming disproportionate resources, even if they distribute requests
// across multiple IPs.
//
// When the limit is exceeded it returns 429 Too Many Requests.
func UserRateLimitMiddleware(limiter ports.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(string(UserIDKey))
		if !exists {
			// This middleware runs after AuthMiddleware, so the userID
			// must always be present. If it's missing, something broke
			// in the middleware chain — fail loudly.
			slog.Error("user rate limiter: userID missing from context",
				"path", c.Request.URL.Path,
			)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		userID, ok := value.(uuid.UUID)
		if !ok {
			slog.Error("user rate limiter: userID has unexpected type",
				"path", c.Request.URL.Path,
			)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		if !limiter.Allow("user:" + userID.String()) {
			slog.Warn("rate limit exceeded",
				"type", "user",
				"user_id", userID.String(),
				"path", c.Request.URL.Path,
			)

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
			})
			return
		}

		c.Next()
	}
}
