package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// --- Database ---
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

	// --- Health checker ---
	healthChecker := persistence.NewPostgresHealthChecker(db)

	// --- HTTP server ---
	router := httpAdapter.SetupRoutes(healthChecker, userService, clientService, authService, tokenProvider)

	port := os.Getenv("APP_PORT")
	if port == "" {
		slog.Error("APP_PORT is not set — check your .env file")
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in a goroutine so it doesn't block
	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// --- Graceful shutdown ---
	// Wait for interrupt signal (Ctrl+C) or SIGTERM (docker stop)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	// Give in-flight requests 10 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	// Close database connection
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	slog.Info("server stopped gracefully")
}
