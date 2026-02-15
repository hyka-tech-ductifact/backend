package main

import (
	"log"
	"os"

	"backend-go-blueprint/internal/application/usecases"
	"backend-go-blueprint/internal/infrastructure/adapters/out/database"
	"backend-go-blueprint/internal/interfaces/http"
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

	userRepo := database.NewPostgresUserRepository(db)
	blueprintUseCase := usecases.NewBlueprintUseCase(userRepo)

	router := http.SetupRoutes(blueprintUseCase)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}