package http_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	handler "ductifact/internal/infrastructure/adapters/inbound/http"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── GET /docs ───────────────────────────────────────────────

func TestDocsHandler_UI_Returns200_WithHTML(t *testing.T) {
	r := gin.New()
	h := handler.NewDocsHandler()
	r.GET("/docs", h.UI)

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Body.String(), "swagger-ui")
	assert.Contains(t, w.Body.String(), "/docs/openapi.yaml")
}

// ─── GET /docs/openapi.yaml ─────────────────────────────────

func TestDocsHandler_Spec_Returns200_WhenSpecExists(t *testing.T) {
	// Create a temporary spec file in a known location
	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "contracts", "openapi")
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	specContent := "openapi: 3.0.3\ninfo:\n  title: Test\n  version: 0.0.1\n"
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "bundled.yaml"), []byte(specContent), 0o644))

	// Change working directory to tmpDir so the handler finds the spec
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	r := gin.New()
	h := handler.NewDocsHandler()
	r.GET("/docs/openapi.yaml", h.Spec)

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/yaml")
	assert.Equal(t, specContent, w.Body.String())
}

func TestDocsHandler_Spec_Returns503_WhenSpecMissing(t *testing.T) {
	// Change to a directory where no spec exists
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	r := gin.New()
	h := handler.NewDocsHandler()
	r.GET("/docs/openapi.yaml", h.Spec)

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "not available")
}
