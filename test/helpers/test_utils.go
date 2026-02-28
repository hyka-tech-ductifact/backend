package helpers

import (
	"fmt"
	"os"
	"testing"

	"event-service/internal/application/services"
	httpAdapter "event-service/internal/infrastructure/adapters/inbound/http"
	"event-service/internal/infrastructure/adapters/outbound/persistence"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SetupTestDB creates a PostgreSQL database connection for testing
func SetupTestDB(t *testing.T) *gorm.DB {
	// Get database configuration from environment variables
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}

	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "microservice_db"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		host, user, password, dbname, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Auto migrate the schema with production models
	err = db.AutoMigrate(&persistence.EventModel{})
	require.NoError(t, err)

	return db
}

// SetupTestRouter creates a test router with in-memory database
func SetupTestRouter(t *testing.T) *gin.Engine {
	db := SetupTestDB(t)

	// Outbound adapter
	eventRepo := persistence.NewPostgresEventRepository(db)

	// Application service (inbound port)
	eventService := services.NewEventService(eventRepo)

	// Inbound adapter: HTTP router
	router := httpAdapter.SetupRoutes(eventService)

	return router
}
