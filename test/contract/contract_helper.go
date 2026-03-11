// Package contract provides contract testing utilities that validate
// API responses directly against the OpenAPI spec (openapi.yaml).
//
// Unlike manual schema checks, this package loads the actual spec file
// and uses kin-openapi to validate every response — so if the spec changes
// and the API doesn't (or vice versa), tests fail.
package contract

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	legacyrouter "github.com/getkin/kin-openapi/routers/legacy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ContractValidator validates API responses against the OpenAPI spec.
// It loads the spec once, builds a router to match requests to operations,
// and validates each response body against the schemas defined in the YAML.
type ContractValidator struct {
	t       *testing.T
	doc     *openapi3.T
	router  routers.Router
	tracker *CoverageTracker
}

// NewContractValidator loads the OpenAPI spec from specPath and creates a
// router for matching requests to operations.
// serverBaseURL must include the API prefix, e.g. "http://localhost:8080/api/v1".
// It overrides the spec's servers list so route matching works with the actual
// test server URL.
func NewContractValidator(t *testing.T, specPath, serverBaseURL string, tracker *CoverageTracker) *ContractValidator {
	t.Helper()

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(specPath)
	require.NoError(t, err, "cannot load OpenAPI spec at: %s", specPath)

	err = doc.Validate(loader.Context)
	require.NoError(t, err, "OpenAPI spec is invalid: %s", specPath)

	// Override servers to match the actual test server URL.
	// This ensures FindRoute matches requests regardless of the host/port
	// defined in the spec file.
	doc.Servers = openapi3.Servers{
		{URL: serverBaseURL},
	}

	router, err := legacyrouter.NewRouter(doc)
	require.NoError(t, err, "cannot create router from OpenAPI spec")

	return &ContractValidator{
		t:       t,
		doc:     doc,
		router:  router,
		tracker: tracker,
	}
}

// ValidateResponse checks that an HTTP response matches the OpenAPI spec.
//
// It validates:
//  1. Status code matches expectedStatus
//  2. The response body conforms to the schema defined in the spec
//     for this operation + status code (required fields, types, enums, etc.)
//  3. The status code is documented in the spec for this operation
//
// This is the ONLY validation method needed — it handles objects, arrays,
// empty bodies (204), and error responses automatically based on the spec.
func (cv *ContractValidator) ValidateResponse(resp *http.Response, expectedStatus int) {
	cv.t.Helper()

	// 1. Assert the status code is what the test expects
	assert.Equal(cv.t, expectedStatus, resp.StatusCode,
		"status code mismatch for %s %s", resp.Request.Method, resp.Request.URL.Path)

	// 2. Buffer the response body (we need to pass it to the validator)
	body, err := io.ReadAll(resp.Body)
	require.NoError(cv.t, err)
	resp.Body.Close()

	// 3. Find the matching operation in the spec
	route, pathParams, err := cv.router.FindRoute(resp.Request)
	require.NoError(cv.t, err,
		"no matching route in OpenAPI spec for %s %s",
		resp.Request.Method, resp.Request.URL)

	// 4. Build validation inputs
	requestInput := &openapi3filter.RequestValidationInput{
		Request:    resp.Request,
		PathParams: pathParams,
		Route:      route,
		Options: &openapi3filter.Options{
			// Skip auth validation — contract tests only check response shape
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
	}

	responseInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: requestInput,
		Status:                 resp.StatusCode,
		Header:                 resp.Header,
		Options: &openapi3filter.Options{
			// Fail if the status code is not documented in the spec
			IncludeResponseStatus: true,
		},
	}

	if len(body) > 0 {
		responseInput.Body = io.NopCloser(bytes.NewReader(body))
	}

	// 5. Validate the response against the OpenAPI spec
	err = openapi3filter.ValidateResponse(context.Background(), responseInput)
	assert.NoError(cv.t, err,
		"response does not match OpenAPI spec for %s %s (status %d):\n%s",
		resp.Request.Method, resp.Request.URL.Path, resp.StatusCode, string(body))

	// 6. Record this operation+status for coverage tracking
	if cv.tracker != nil {
		cv.tracker.Record(resp.Request.Method, route.Path, resp.StatusCode)
	}
}

// --- Helpers ---

// DefaultSpecPath returns the path to the OpenAPI spec relative to the test directory.
func DefaultSpecPath() string {
	paths := []string{
		"../../../contracts/openapi/bundled.yaml", // from backend/test/contract/
		"../../contracts/openapi/bundled.yaml",    // fallback
		"../contracts/openapi/bundled.yaml",       // from backend/
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fallback: environment variable
	if envPath := os.Getenv("CONTRACT_SPEC_PATH"); envPath != "" {
		return envPath
	}

	return "../../../contracts/openapi/bundled.yaml"
}

// ═══════════════════════════════════════════════════════════════════════════════
// Coverage Tracker
// ═══════════════════════════════════════════════════════════════════════════════

// CoverageTracker tracks which operation+status combinations from the OpenAPI
// spec have been validated by contract tests. After running all tests, call
// Missing() to find undocumented gaps — if any endpoint or status code defined
// in the spec was NOT exercised, it means the contract is not fully verified.
type CoverageTracker struct {
	mu       sync.Mutex
	covered  map[string]bool // e.g. "GET /health → 200"
	expected map[string]bool // all operation+status from spec
}

// NewCoverageTracker loads the OpenAPI spec and builds the set of all
// expected operation+status combinations that must be tested.
func NewCoverageTracker(specPath string) (*CoverageTracker, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("coverage tracker: cannot load spec: %w", err)
	}

	expected := make(map[string]bool)
	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			if operation.Responses == nil {
				continue
			}
			for statusCode := range operation.Responses.Map() {
				// Skip "default" and wildcard responses (e.g. "2XX")
				if _, err := strconv.Atoi(statusCode); err != nil {
					continue
				}
				key := fmt.Sprintf("%s %s → %s", method, path, statusCode)
				expected[key] = true
			}
		}
	}

	return &CoverageTracker{
		covered:  make(map[string]bool),
		expected: expected,
	}, nil
}

// Record marks an operation+status as covered by a test.
func (ct *CoverageTracker) Record(method, pathPattern string, status int) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	key := fmt.Sprintf("%s %s → %d", method, pathPattern, status)
	ct.covered[key] = true
}

// Exclude removes an operation+status from the expected set.
// Use this for operations that cannot be tested from contract tests
// (e.g. health 503 requires killing the database).
func (ct *CoverageTracker) Exclude(method, path string, status int) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	key := fmt.Sprintf("%s %s → %d", method, path, status)
	delete(ct.expected, key)
}

// Missing returns the list of expected operation+status combinations
// that have NOT been validated by any test.
func (ct *CoverageTracker) Missing() []string {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	var missing []string
	for key := range ct.expected {
		if !ct.covered[key] {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing
}

// Report returns a human-readable summary of contract coverage.
func (ct *CoverageTracker) Report() string {
	ct.mu.Lock()
	total := len(ct.expected)
	covered := 0
	for key := range ct.expected {
		if ct.covered[key] {
			covered++
		}
	}
	ct.mu.Unlock()

	pct := 0.0
	if total > 0 {
		pct = float64(covered) / float64(total) * 100
	}
	return fmt.Sprintf("Contract coverage: %d/%d operations validated (%.0f%%)", covered, total, pct)
}
