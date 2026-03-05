package main

import (
	"log"
	"os"

	"ductifact/internal/application/services"
	httpAdapter "ductifact/internal/infrastructure/adapters/inbound/http"
	"ductifact/internal/infrastructure/adapters/outbound/persistence"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (ignored if not found, e.g. in Docker/CI)
	_ = godotenv.Load()

	// Database
	db, err := persistence.NewPostgresConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// --- User wiring ---
	userRepo := persistence.NewPostgresUserRepository(db)
	userService := services.NewUserService(userRepo)

	// --- Client wiring ---
	clientRepo := persistence.NewPostgresClientRepository(db)
	clientService := services.NewClientService(clientRepo, userRepo)

	// --- HTTP ---
	router := httpAdapter.SetupRoutes(userService, clientService)

	// ...

	port := os.Getenv("APP_PORT")
	if port == "" {
		log.Fatal("APP_PORT is not set — check your .env file")
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
