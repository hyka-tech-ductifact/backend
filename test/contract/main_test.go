package contract_test

import (
	"fmt"
	"os"
	"testing"

	"ductifact/internal/config"
	"ductifact/test/contract"
	"ductifact/test/helpers"

	"gorm.io/gorm"
)

// contractEnv holds the shared state for contract tests.
type contractEnv struct {
	baseURL string
	db      *gorm.DB
	tracker *contract.CoverageTracker
}

// env is the shared contract test environment.
var env *contractEnv

// TestMain runs ONCE before all contract tests. Requires DB + API running.
func TestMain(m *testing.M) {
	helpers.LoadEnv()

	cfg := config.Load()
	baseURL := "http://" + cfg.App.Host + ":" + cfg.App.Port

	if err := helpers.WaitForAPI(baseURL, 10); err != nil {
		fmt.Printf("Contract test setup failed: %v\n", err)
		os.Exit(1)
	}

	db, err := helpers.ConnectTestDB()
	if err != nil {
		fmt.Printf("Contract DB setup failed: %v\n", err)
		os.Exit(1)
	}

	specPath := contract.DefaultSpecPath()
	tracker, err := contract.NewCoverageTracker(specPath)
	if err != nil {
		fmt.Printf("Coverage tracker setup failed: %v\n", err)
		os.Exit(1)
	}

	// Exclude operations that cannot be tested from contract tests.
	// GET /health → 503 requires the database to be down, which is not
	// feasible when the API server manages its own DB pool.
	// This scenario is already covered by unit tests (health_handler_test.go).
	tracker.Exclude("GET", "/health", 503)
	// GET /health → 404 is a generic spec response for "endpoint not found".
	// It cannot be triggered on an existing endpoint.
	tracker.Exclude("GET", "/health", 404)

	env = &contractEnv{
		baseURL: baseURL,
		db:      db,
		tracker: tracker,
	}

	code := m.Run()

	// --- Contract coverage report ---
	fmt.Println("\n" + tracker.Report())
	if missing := tracker.Missing(); len(missing) > 0 {
		fmt.Println("\nUNTESTED contract operations (spec defines them but no test validates them):")
		for _, op := range missing {
			fmt.Println("  ✗", op)
		}
		if code == 0 {
			code = 1 // fail the suite if coverage is incomplete
		}
	}

	os.Exit(code)
}

// clean truncates all tables before each test to ensure isolation.
func clean(t *testing.T) {
	t.Helper()
	helpers.CleanDB(t, env.db)
}

// url builds a full URL: baseURL + /v1 + path (for versioned endpoints).
func url(path string) string {
	return fmt.Sprintf("%s/v1%s", env.baseURL, path)
}

// rootURL builds a full URL: baseURL + path (for unversioned endpoints like /health).
func rootURL(path string) string {
	return fmt.Sprintf("%s%s", env.baseURL, path)
}

// newValidator creates a ContractValidator for the current test.
func newValidator(t *testing.T) *contract.ContractValidator {
	t.Helper()
	specPath := contract.DefaultSpecPath()
	return contract.NewContractValidator(t, specPath, env.baseURL, env.tracker)
}

// authTokens holds both access and refresh tokens returned by register/login.
type authTokens struct {
	AccessToken  string
	RefreshToken string
}

// registerAndLogin creates a user via /auth/register and returns both tokens.
// This is a test helper — it does NOT validate the contract, just obtains tokens.
func registerAndLogin(t *testing.T, name, email, password string) authTokens {
	t.Helper()

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     name,
		"email":    email,
		"password": password,
	})

	body := helpers.ParseBody(t, resp)
	access, ok := body["access_token"].(string)
	if !ok || access == "" {
		t.Fatalf("registerAndLogin: expected access_token in response, got: %v", body)
	}
	refresh, ok := body["refresh_token"].(string)
	if !ok || refresh == "" {
		t.Fatalf("registerAndLogin: expected refresh_token in response, got: %v", body)
	}
	return authTokens{AccessToken: access, RefreshToken: refresh}
}
