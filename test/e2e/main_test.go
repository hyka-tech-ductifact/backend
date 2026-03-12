package e2e

import (
	"fmt"
	"os"
	"testing"

	"ductifact/internal/config"
	"ductifact/test/helpers"

	"gorm.io/gorm"
)

// e2eEnv holds the shared state for E2E tests, initialized once in TestMain.
type e2eEnv struct {
	baseURL string
	db      *gorm.DB
}

// env is the shared E2E environment, visible to ALL _test.go files in this package.
var env *e2eEnv

// TestMain runs ONCE before all tests in this package, regardless of how many
// _test.go files exist. It loads env vars, waits for the API, and connects to the DB.
func TestMain(m *testing.M) {
	helpers.LoadEnv()

	cfg := config.Load()
	baseURL := "http://" + cfg.App.Host + ":" + cfg.App.Port

	if err := helpers.WaitForAPI(baseURL, 10); err != nil {
		fmt.Printf("E2E setup failed: %v\n", err)
		os.Exit(1)
	}

	db, err := helpers.ConnectTestDB()
	if err != nil {
		fmt.Printf("E2E DB setup failed: %v\n", err)
		os.Exit(1)
	}

	env = &e2eEnv{
		baseURL: baseURL,
		db:      db,
	}

	os.Exit(m.Run())
}

// clean truncates all tables before each test to ensure isolation.
func clean(t *testing.T) {
	t.Helper()
	helpers.CleanDB(t, env.db)
}

// url builds a full URL using the shared base URL.
func url(path string) string {
	return fmt.Sprintf("%s/api/v1%s", env.baseURL, path)
}
