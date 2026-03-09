package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoveryMiddleware_CatchesPanic_Returns500(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	r.GET("/panic", func(c *gin.Context) {
		panic("something went terribly wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()

	// This should NOT panic — the middleware catches it
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "internal server error", body["error"])
	assert.NotEmpty(t, body["request_id"])
}

func TestRecoveryMiddleware_NoPanic_PassesThrough(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RecoveryMiddleware())
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRecoveryMiddleware_CatchesNilPointerPanic(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RecoveryMiddleware())
	r.GET("/nil-panic", func(c *gin.Context) {
		var s *string
		_ = *s // nil pointer dereference → panic
	})

	req := httptest.NewRequest(http.MethodGet, "/nil-panic", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "internal server error", body["error"])
}

func TestRecoveryMiddleware_WithoutRequestID_StillWorks(t *testing.T) {
	r := gin.New()
	// No RequestIDMiddleware — recovery should still work
	r.Use(middleware.RecoveryMiddleware())
	r.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "internal server error", body["error"])
	assert.Empty(t, body["request_id"]) // No request ID middleware → empty
}
