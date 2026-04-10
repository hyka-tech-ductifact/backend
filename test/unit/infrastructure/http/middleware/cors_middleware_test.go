package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupCORSRouter creates a test router using middleware.CORSMiddleware.
// allowLocalhost controls whether http://localhost on any port is permitted.
func setupCORSRouter(allowLocalhost bool) *gin.Engine {
	origins := []string{"https://app.example.com"}
	if allowLocalhost {
		origins = append(origins, "http://localhost")
	}

	r := gin.New()
	r.Use(middleware.CORSMiddleware(origins))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	return r
}

func TestCORS_ConfiguredOrigin_ReturnsHeaders(t *testing.T) {
	r := setupCORSRouter(false)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_LocalhostAllowed_ReturnsHeaders(t *testing.T) {
	r := setupCORSRouter(true)

	for _, origin := range []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:8080"} {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "origin %s should be allowed", origin)
		assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"), "origin %s", origin)
	}
}

func TestCORS_LocalhostBlocked_WhenNotConfigured(t *testing.T) {
	r := setupCORSRouter(false)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_DisallowedOrigin_NoHeaders(t *testing.T) {
	r := setupCORSRouter(true)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://evil-site.com")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// gin-contrib/cors blocks requests from disallowed origins with 403.
	// In a real browser, the preflight would fail first. This tests server-side enforcement.
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_PreflightRequest_Returns204(t *testing.T) {
	r := setupCORSRouter(true)

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
}

func TestCORS_PreflightFromDisallowedOrigin_NoHeaders(t *testing.T) {
	r := setupCORSRouter(true)

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://evil-site.com")
	req.Header.Set("Access-Control-Request-Method", "DELETE")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_ExposesRequestIDHeader(t *testing.T) {
	r := setupCORSRouter(true)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// The browser should be able to read X-Request-ID from the response.
	// Go's HTTP library normalizes header names, so "X-Request-ID" becomes "X-Request-Id".
	exposeHeaders := w.Header().Get("Access-Control-Expose-Headers")
	assert.Contains(t, strings.ToLower(exposeHeaders), strings.ToLower("X-Request-ID"))
}

func TestCORS_AllowsCredentials(t *testing.T) {
	r := setupCORSRouter(true)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}
