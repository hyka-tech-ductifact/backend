package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequestIDMiddleware_GeneratesNewID(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.GET("/test", func(c *gin.Context) {
		requestID := middleware.GetRequestIDFromContext(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Response header should contain the request ID
	headerID := w.Header().Get("X-Request-ID")
	assert.NotEmpty(t, headerID)
	assert.Len(t, headerID, 36) // UUID format: 8-4-4-4-12
}

func TestRequestIDMiddleware_ReusesClientID(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.GET("/test", func(c *gin.Context) {
		requestID := middleware.GetRequestIDFromContext(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	clientID := "my-custom-request-id-123"
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", clientID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Should reuse the client's ID
	headerID := w.Header().Get("X-Request-ID")
	assert.Equal(t, clientID, headerID)
}

func TestRequestIDMiddleware_EachRequestGetsDifferentID(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/test", nil))

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/test", nil))

	id1 := w1.Header().Get("X-Request-ID")
	id2 := w2.Header().Get("X-Request-ID")

	assert.NotEqual(t, id1, id2, "Each request should get a unique ID")
}

func TestGetRequestIDFromContext_NoMiddleware_ReturnsEmpty(t *testing.T) {
	r := gin.New()
	// No RequestIDMiddleware registered
	r.GET("/test", func(c *gin.Context) {
		requestID := middleware.GetRequestIDFromContext(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("X-Request-ID"))
}
