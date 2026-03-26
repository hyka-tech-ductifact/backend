package contract_test

import (
	"net/http"
	"testing"

	"ductifact/test/helpers"
)

// ─── GET /health ─────────────────────────────────────────────

func TestContract_HealthCheck_Returns200_WithHealthResponse(t *testing.T) {
	cv := newValidator(t)

	resp := helpers.GetJSON(t, rootURL("/health"))

	// Validates against HealthResponse schema from openapi.yaml
	cv.ValidateResponse(resp, http.StatusOK)
}
