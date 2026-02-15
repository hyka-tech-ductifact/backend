package main

import (
	"log"
	"os"

	"event-service/internal/application/usecases"
	"event-service/internal/infrastructure/adapters/out/database"
	"event-service/internal/interfaces/http"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	db, err := database.NewPostgresConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	eventRepo := database.NewPostgresEventRepository(db)
	eventUseCase := usecases.NewEventUseCase(eventRepo)

	router := http.SetupRoutes(eventUseCase)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
