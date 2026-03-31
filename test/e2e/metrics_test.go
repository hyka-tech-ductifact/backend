package e2e
package e2e

import (
	"io"
	"net/http"
	"testing"

	"ductifact/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── GET /metrics ────────────────────────────────────────────

func TestE2E_Metrics_Returns200_WithPrometheusMetrics(t *testing.T) {
	resp := helpers.GetJSON(t, rootURL("/metrics"))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	content := string(body)
	assert.Contains(t, content, "http_requests_total")
	assert.Contains(t, content, "http_request_duration_seconds")
}
