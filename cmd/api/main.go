package main

import (
	"log"
	"os"

	"event-service/internal/application/services"
	httpAdapter "event-service/internal/infrastructure/adapters/inbound/http"
	"event-service/internal/infrastructure/adapters/outbound/persistence"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	// Outbound adapter: PostgreSQL
	db, err := persistence.NewPostgresConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Outbound adapter implements domain repository interface (outbound port)
	eventRepo := persistence.NewPostgresEventRepository(db)

	// Application service implements inbound port
	eventService := services.NewEventService(eventRepo)

	// Inbound adapter: HTTP
	router := httpAdapter.SetupRoutes(eventService)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
