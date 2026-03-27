package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
)

// DocsHandler serves Swagger UI and the raw OpenAPI specification.
//
// Swagger UI is rendered from a self-contained HTML page that loads
// assets from the unpkg CDN. The OpenAPI spec is read from disk once
// at first request and cached in memory.
//
// Routes:
//
//	GET /docs              → Swagger UI (interactive documentation)
//	GET /docs/openapi.yaml → raw OpenAPI spec (YAML)
type DocsHandler struct {
	specOnce sync.Once
	specData []byte
	specErr  error
}

// NewDocsHandler creates a handler that serves API documentation.
func NewDocsHandler() *DocsHandler {
	return &DocsHandler{}
}

// specPaths lists the locations where the bundled OpenAPI spec may exist,
// ordered by priority. The first file found is used.
var specPaths = []string{
	"contracts/openapi/bundled.yaml",      // from project root (make dev / make app-start)
	"../contracts/openapi/bundled.yaml",   // from bin/ directory
	"/app/contracts/openapi/bundled.yaml", // Docker container
}

// loadSpec reads the OpenAPI spec from the first available path.
// The result is cached — the file is read only once.
func (h *DocsHandler) loadSpec() ([]byte, error) {
	h.specOnce.Do(func() {
		for _, p := range specPaths {
			data, err := os.ReadFile(p)
			if err == nil {
				h.specData = data
				slog.Info("OpenAPI spec loaded", "path", p)
				return
			}
		}
		h.specErr = fmt.Errorf("OpenAPI spec not found in any of: %v", specPaths)
		slog.Warn("OpenAPI spec not available", "searched", specPaths)
	})
	return h.specData, h.specErr
}

// UI serves the Swagger UI HTML page.
// The page loads Swagger UI assets from the unpkg CDN and points
// to /docs/openapi.yaml for the API specification.
func (h *DocsHandler) UI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
}

// Spec serves the raw OpenAPI specification as YAML.
// Returns 503 if the spec file was not found at startup.
func (h *DocsHandler) Spec(c *gin.Context) {
	data, err := h.loadSpec()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OpenAPI spec not available — run: make fetch-contract",
		})
		return
	}
	c.Data(http.StatusOK, "application/yaml", data)
}

// swaggerHTML is a self-contained page that renders Swagger UI.
// Assets are loaded from the unpkg CDN to keep the binary small.
const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Ductifact API — Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: '/docs/openapi.yaml',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset
      ],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`
