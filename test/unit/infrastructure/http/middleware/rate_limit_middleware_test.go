package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"
	"ductifact/test/unit/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// =============================================================================
// IPRateLimitMiddleware
// =============================================================================

func TestIPRateLimit_Allowed_PassesThrough(t *testing.T) {
	limiter := &mocks.MockRateLimiter{
		AllowFn: func(key string) bool { return true },
	}

	r := gin.New()
	r.Use(middleware.IPRateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRateLimit_Exceeded_Returns429(t *testing.T) {
	limiter := &mocks.MockRateLimiter{
		AllowFn: func(key string) bool { return false },
	}

	r := gin.New()
	r.Use(middleware.IPRateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Contains(t, body["error"], "too many requests")
}

func TestIPRateLimit_UsesIPPrefixedKey(t *testing.T) {
	var capturedKey string
	limiter := &mocks.MockRateLimiter{
		AllowFn: func(key string) bool {
			capturedKey = key
			return true
		},
	}

	r := gin.New()
	r.Use(middleware.IPRateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, capturedKey, "ip:")
}

func TestIPRateLimit_Exceeded_DoesNotCallHandler(t *testing.T) {
	handlerCalled := false

	limiter := &mocks.MockRateLimiter{
		AllowFn: func(key string) bool { return false },
	}

	r := gin.New()
	r.Use(middleware.IPRateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.False(t, handlerCalled)
}

// =============================================================================
// UserRateLimitMiddleware
// =============================================================================

func TestUserRateLimit_Allowed_PassesThrough(t *testing.T) {
	userID := uuid.New()
	limiter := &mocks.MockRateLimiter{
		AllowFn: func(key string) bool { return true },
	}

	r := gin.New()
	// Simulate auth middleware setting user ID
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	r.Use(middleware.UserRateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserRateLimit_Exceeded_Returns429(t *testing.T) {
	userID := uuid.New()
	limiter := &mocks.MockRateLimiter{
		AllowFn: func(key string) bool { return false },
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	r.Use(middleware.UserRateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestUserRateLimit_UsesUserPrefixedKey(t *testing.T) {
	userID := uuid.New()
	var capturedKey string
	limiter := &mocks.MockRateLimiter{
		AllowFn: func(key string) bool {
			capturedKey = key
			return true
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	r.Use(middleware.UserRateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "user:"+userID.String(), capturedKey)
}

func TestUserRateLimit_NoUserInContext_Returns500(t *testing.T) {
	limiter := &mocks.MockRateLimiter{
		AllowFn: func(key string) bool { return true },
	}

	r := gin.New()
	// No auth middleware — no userID in context
	r.Use(middleware.UserRateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "internal server error", body["error"])
}

func TestUserRateLimit_Exceeded_DoesNotCallHandler(t *testing.T) {
	userID := uuid.New()
	handlerCalled := false

	limiter := &mocks.MockRateLimiter{
		AllowFn: func(key string) bool { return false },
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	r.Use(middleware.UserRateLimitMiddleware(limiter))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.False(t, handlerCalled)
}
