package main

import (
	"log/slog"
	"os"

	"ductifact/internal/application/services"
	httpAdapter "ductifact/internal/infrastructure/adapters/inbound/http"
	"ductifact/internal/infrastructure/adapters/outbound/persistence"
	"ductifact/internal/infrastructure/auth"
	"ductifact/internal/infrastructure/logging"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (ignored if not found, e.g. in Docker/CI)
	_ = godotenv.Load()

	// --- Logger ---
	logger := logging.NewLogger()
	slog.SetDefault(logger)

	// Database
	db, err := persistence.NewPostgresConnection()
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// --- User wiring ---
	userRepo := persistence.NewPostgresUserRepository(db)
	userService := services.NewUserService(userRepo)

	// --- Client wiring ---
	clientRepo := persistence.NewPostgresClientRepository(db)
	clientService := services.NewClientService(clientRepo, userRepo)

	// --- Auth wiring ---
	tokenProvider := auth.NewJWTProvider()
	authService := services.NewAuthService(userRepo, tokenProvider)

	// --- HTTP ---
	router := httpAdapter.SetupRoutes(userService, clientService, authService, tokenProvider)

	port := os.Getenv("APP_PORT")
	if port == "" {
		slog.Error("APP_PORT is not set — check your .env file")
		os.Exit(1)
	}

	slog.Info("server starting", "port", port)
	if err := router.Run(":" + port); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
