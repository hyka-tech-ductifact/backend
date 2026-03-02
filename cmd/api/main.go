package main

import (
	"log"
	"os"

	"event-service/internal/application/services"
	httpAdapter "event-service/internal/infrastructure/adapters/inbound/http"
	"event-service/internal/infrastructure/adapters/outbound/persistence"
)

func main() {
	// ...

	// Database
	db, err := persistence.NewPostgresConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// --- User wiring ---
	userRepo := persistence.NewPostgresUserRepository(db)
	userService := services.NewUserService(userRepo)

	// --- HTTP ---
	router := httpAdapter.SetupRoutes(userService)

	// ...

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
