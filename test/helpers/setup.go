package helpers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"ductifact/internal/infrastructure/adapters/outbound/persistence"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// LoadEnv loads the .env file from the project root.
// It silently does nothing if the file doesn't exist (e.g. in CI where env vars are set externally).
func LoadEnv() {
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	_ = godotenv.Load(filepath.Join(projectRoot, ".env"))
}

// SetupTestDB creates a PostgreSQL database connection for testing.
// Reads credentials from the .env file at the project root.
func SetupTestDB(t *testing.T) *gorm.DB {
	LoadEnv()

	db, err := ConnectTestDB()
	require.NoError(t, err, "failed to connect to test DB")

	return db
}

// ConnectTestDB creates a PostgreSQL connection using env vars.
// Returns an error instead of calling t.Fatal — safe for use in TestMain.
func ConnectTestDB() (*gorm.DB, error) {
	requiredVars := []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME"}
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			return nil, fmt.Errorf("%s is not set — check your .env file", v)
		}
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}

	if err := db.AutoMigrate(&persistence.UserModel{}, &persistence.ClientModel{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	return db, nil
}

// CleanDB truncates all tables to ensure test isolation.
// Call this at the beginning of each integration test.
// Order matters: clients references users, so truncate clients first.
func CleanDB(t *testing.T, db *gorm.DB) {
	err := db.Exec("TRUNCATE TABLE clients, users RESTART IDENTITY CASCADE").Error
	require.NoError(t, err)
}

// WaitForAPI polls baseURL/api/v1/health until the API responds 200 or retries are exhausted.
// Use this in TestMain to block until the server is ready.
func WaitForAPI(baseURL string, maxRetries int) error {
	for i := range maxRetries {
		resp, err := http.Get(baseURL + "/api/v1/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if i == maxRetries-1 {
			return fmt.Errorf("API not ready at %s after %d retries — is the server running? (make app-run)", baseURL, maxRetries)
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("API not ready at %s", baseURL)
}
