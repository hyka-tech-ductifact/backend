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
	"ductifact/internal/config"
	httpAdapter "ductifact/internal/infrastructure/adapters/inbound/http"
	"ductifact/internal/infrastructure/adapters/outbound/persistence"
	"ductifact/internal/infrastructure/auth"
	"ductifact/internal/infrastructure/logging"
	"ductifact/internal/infrastructure/ratelimit"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (ignored if not found, e.g. in Docker/CI)
	_ = godotenv.Load()

	// --- Configuration (panics on missing required vars) ---
	cfg := config.Load()

	// --- Logger ---
	logger := logging.NewLogger(cfg.Log)
	slog.SetDefault(logger)

	// --- Database ---
	db, err := persistence.NewPostgresConnection(cfg.Database, cfg.Log.Level)
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

	// --- Project wiring ---
	projectRepo := persistence.NewPostgresProjectRepository(db)
	projectService := services.NewProjectService(projectRepo, clientRepo)

	// --- Auth wiring ---
	tokenProvider := auth.NewJWTProvider(cfg.JWT)
	blacklist := auth.NewMemoryBlacklist(5 * time.Minute)
	defer blacklist.Stop()

	// --- Login throttler ---
	loginThrottler := ratelimit.NewMemoryLoginThrottler(
		cfg.LoginThrottle.MaxAttempts,
		cfg.LoginThrottle.Window,
		cfg.LoginThrottle.LockoutDuration,
		1*time.Minute,
	)
	defer loginThrottler.Stop()

	authService := services.NewAuthService(userRepo, tokenProvider, blacklist, loginThrottler, cfg.JWT.TokenDuration, cfg.JWT.RefreshTokenDuration)

	// --- Health checker ---
	healthChecker := persistence.NewPostgresHealthChecker(db)

	// --- Rate limiters ---
	ipLimiter := ratelimit.NewMemoryRateLimiter(
		cfg.RateLimit.IPMaxRequests,
		cfg.RateLimit.IPWindow,
		1*time.Minute,
	)
	defer ipLimiter.Stop()

	userLimiter := ratelimit.NewMemoryRateLimiter(
		cfg.RateLimit.UserMaxRequests,
		cfg.RateLimit.UserWindow,
		1*time.Minute,
	)
	defer userLimiter.Stop()

	// --- HTTP server ---
	router := httpAdapter.SetupRoutes(healthChecker, userService, clientService, projectService, authService, tokenProvider, blacklist, ipLimiter, userLimiter, cfg.CORS)

	port := cfg.App.Port
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
