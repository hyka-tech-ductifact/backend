package helpers

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

// LoadTestEnv loads environment variables for testing
func LoadTestEnv(t *testing.T) {
	// Try to load .env file, but don't fail if it doesn't exist
	_ = godotenv.Load()

	// Set test-specific environment variables
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "test_user")
	os.Setenv("DB_PASSWORD", "test_password")
	os.Setenv("DB_NAME", "event_service_test")
	os.Setenv("PORT", "8080")
}

// TestConfig holds test configuration
type TestConfig struct {
	BaseURL string
	DBURL   string
}

// GetTestConfig returns test configuration
func GetTestConfig() *TestConfig {
	return &TestConfig{
		BaseURL: "http://localhost:8080",
		DBURL:   "postgres://test_user:test_password@localhost:5432/event_service_test?sslmode=disable",
	}
}
