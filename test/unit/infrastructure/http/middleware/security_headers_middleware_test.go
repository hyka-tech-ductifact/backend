package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// SecurityHeadersMiddleware — header presence
// =============================================================================

func TestSecurityHeaders_SetsStrictTransportSecurity(t *testing.T) {
	w := performSecurityHeadersRequest(t)

	assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeaders_SetsXContentTypeOptions(t *testing.T) {
	w := performSecurityHeadersRequest(t)

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}

func TestSecurityHeaders_SetsXFrameOptions(t *testing.T) {
	w := performSecurityHeadersRequest(t)

	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
}

func TestSecurityHeaders_SetsStrictCSPForAPIRoutes(t *testing.T) {
	w := performSecurityHeadersRequest(t)

	assert.Equal(t, "default-src 'self'; frame-ancestors 'none'", w.Header().Get("Content-Security-Policy"))
}

func TestSecurityHeaders_SetsRelaxedCSPForDocsRoute(t *testing.T) {
	r := gin.New()
	r.Use(middleware.SecurityHeadersMiddleware())
	r.GET("/docs", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "https://unpkg.com")
	assert.Contains(t, csp, "'unsafe-inline'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
}

func TestSecurityHeaders_SetsRelaxedCSPForDocsSubpath(t *testing.T) {
	r := gin.New()
	r.Use(middleware.SecurityHeadersMiddleware())
	r.GET("/docs/openapi.yaml", func(c *gin.Context) {
		c.String(http.StatusOK, "openapi: 3.0.3")
	})

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "https://unpkg.com")
}

func TestSecurityHeaders_SetsReferrerPolicy(t *testing.T) {
	w := performSecurityHeadersRequest(t)

	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
}

func TestSecurityHeaders_SetsXXSSProtection(t *testing.T) {
	w := performSecurityHeadersRequest(t)

	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
}

func TestSecurityHeaders_SetsPermissionsPolicy(t *testing.T) {
	w := performSecurityHeadersRequest(t)

	assert.Equal(t, "camera=(), microphone=(), geolocation=()", w.Header().Get("Permissions-Policy"))
}

// =============================================================================
// SecurityHeadersMiddleware — behavior
// =============================================================================

func TestSecurityHeaders_PassesThrough(t *testing.T) {
	w := performSecurityHeadersRequest(t)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSecurityHeaders_PresentOnErrorResponses(t *testing.T) {
	r := gin.New()
	r.Use(middleware.SecurityHeadersMiddleware())
	r.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "boom"})
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}

func TestSecurityHeaders_PresentOn404(t *testing.T) {
	r := gin.New()
	r.Use(middleware.SecurityHeadersMiddleware())
	// No routes registered → 404

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
}

// =============================================================================
// Helper
// =============================================================================

func performSecurityHeadersRequest(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()

	r := gin.New()
	r.Use(middleware.SecurityHeadersMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}
