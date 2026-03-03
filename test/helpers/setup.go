package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"ductifact/internal/application/services"
	httpAdapter "ductifact/internal/infrastructure/adapters/inbound/http"
	"ductifact/internal/infrastructure/adapters/outbound/persistence"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// loadEnv loads the .env file from the project root.
// It silently does nothing if the file doesn't exist (e.g. in CI where env vars are set externally).
func loadEnv() {
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	_ = godotenv.Load(filepath.Join(projectRoot, ".env"))
}

// SetupTestDB creates a PostgreSQL database connection for testing.
// Reads credentials from the .env file at the project root.
func SetupTestDB(t *testing.T) *gorm.DB {
	loadEnv()

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	require.NotEmpty(t, host, "DB_HOST is not set — check your .env file")
	require.NotEmpty(t, port, "DB_PORT is not set — check your .env file")
	require.NotEmpty(t, user, "DB_USER is not set — check your .env file")
	require.NotEmpty(t, password, "DB_PASSWORD is not set — check your .env file")
	require.NotEmpty(t, dbname, "DB_NAME is not set — check your .env file")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		host, user, password, dbname, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Auto migrate the schema with production models
	err = db.AutoMigrate(&persistence.UserModel{})
	require.NoError(t, err)

	return db
}

// CleanDB truncates all tables to ensure test isolation.
// Call this at the beginning of each integration test.
func CleanDB(t *testing.T, db *gorm.DB) {
	err := db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE").Error
	require.NoError(t, err)
}

// SetupTestRouter creates a test router with in-memory database
func SetupTestRouter(t *testing.T) *gin.Engine {
	db := SetupTestDB(t)

	// Outbound adapter
	userRepo := persistence.NewPostgresUserRepository(db)

	// Application service (inbound port)
	userService := services.NewUserService(userRepo)

	// Inbound adapter: HTTP router
	router := httpAdapter.SetupRoutes(userService)

	return router
}
