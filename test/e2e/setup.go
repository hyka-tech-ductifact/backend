package e2e

import (
	"net/http"
	"os"
	"testing"
	"time"
)

type TestEnvironment struct {
	APIBaseURL string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	Cleanup    func()
}

func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	// Check if we're running in Docker (use existing containers)
	if os.Getenv("DB_HOST") == "test-db" {
		return setupDockerEnvironment(t)
	}

	// Otherwise, use local environment
	return setupLocalEnvironment(t)
}

func setupDockerEnvironment(t *testing.T) *TestEnvironment {
	// When running in Docker, connect to the API container
	apiBaseURL := "http://test-api:8080"
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(apiBaseURL + "/health")
		if err == nil && resp.StatusCode == 200 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	return &TestEnvironment{
		APIBaseURL: apiBaseURL,
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		Cleanup:    func() {}, // No cleanup needed for Docker environment
	}
}

func setupLocalEnvironment(t *testing.T) *TestEnvironment {
	// For local development, assume services are running on default ports
	apiBaseURL := "http://localhost:8080"

	// Wait for API to be ready
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(apiBaseURL + "/health")
		if err == nil && resp.StatusCode == 200 {
			break
		}
		if i == maxRetries-1 {
			t.Fatalf("API not ready after %d retries. Make sure the API is running on %s", maxRetries, apiBaseURL)
		}
		time.Sleep(1 * time.Second)
	}

	return &TestEnvironment{
		APIBaseURL: apiBaseURL,
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "postgres",
		DBPassword: "postgres123",
		DBName:     "microservice_db",
		Cleanup:    func() {}, // No cleanup needed for local environment
	}
}

// WaitForAPIReady waits for the API to be ready
func WaitForAPIReady(t *testing.T, baseURL string, maxRetries int) {
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == 200 {
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatalf("API not ready after %d retries", maxRetries)
}
