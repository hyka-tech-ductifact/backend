package middleware_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"ductifact/internal/infrastructure/adapters/inbound/http/helpers"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test errors (not real domain errors — just for testing)
var (
	errTestNotFound = errors.New("test: not found")
	errTestConflict = errors.New("test: conflict")
)

func TestHandleError_RegisteredError_ReturnsMappedStatus(t *testing.T) {
	// Register test errors
	helpers.RegisterDomainError(errTestNotFound, http.StatusNotFound, "resource not found")
	helpers.RegisterDomainError(errTestConflict, http.StatusConflict, "resource conflict")

	r := gin.New()
	r.GET("/test-not-found", func(c *gin.Context) {
		helpers.HandleError(c, errTestNotFound)
	})
	r.GET("/test-conflict", func(c *gin.Context) {
		helpers.HandleError(c, errTestConflict)
	})

	// Test 404
	req := httptest.NewRequest(http.MethodGet, "/test-not-found", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "resource not found", body["error"])

	// Test 409
	req = httptest.NewRequest(http.MethodGet, "/test-conflict", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var bodyConflict map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &bodyConflict)
	require.NoError(t, err)
	assert.Equal(t, "resource conflict", bodyConflict["error"])
}

func TestHandleError_UnknownError_Returns500(t *testing.T) {
	unknownErr := errors.New("some unexpected database error")

	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		helpers.HandleError(c, unknownErr)
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

func TestHandleError_WrappedError_StillMatches(t *testing.T) {
	// errors.Is works with wrapped errors thanks to Go's error wrapping
	helpers.RegisterDomainError(errTestNotFound, http.StatusNotFound, "resource not found")

	wrappedErr := fmt.Errorf("service layer: %w", errTestNotFound)

	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		helpers.HandleError(c, wrappedErr)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "resource not found", body["error"])
}

func TestHandleError_NilError_DoesNotPanic(t *testing.T) {
	// Edge case: passing nil should return 500 (unknown error)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		helpers.HandleError(c, errors.New(""))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
