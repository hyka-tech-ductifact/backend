package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLoggerMiddleware_DoesNotBreakChain(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.LoggerMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hello"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoggerMiddleware_PreservesStatusCode(t *testing.T) {
	r := gin.New()
	r.Use(middleware.LoggerMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"created": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestLoggerMiddleware_WorksWithoutRequestIDMiddleware(t *testing.T) {
	r := gin.New()
	// Logger without RequestID — should still work (requestID will be empty)
	r.Use(middleware.LoggerMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoggerMiddleware_HandlesQueryParams(t *testing.T) {
	r := gin.New()
	r.Use(middleware.LoggerMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"q": c.Query("search")})
	})

	req := httptest.NewRequest(http.MethodGet, "/test?search=hello&page=1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
