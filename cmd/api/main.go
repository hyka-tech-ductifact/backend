package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ductifact/internal/application/ports"
	"ductifact/internal/application/services"
	"ductifact/internal/config"
	httpAdapter "ductifact/internal/infrastructure/adapters/inbound/http"
	"ductifact/internal/infrastructure/adapters/outbound/email"
	"ductifact/internal/infrastructure/adapters/outbound/imageproc"
	"ductifact/internal/infrastructure/adapters/outbound/persistence"
	"ductifact/internal/infrastructure/adapters/outbound/storage"
	"ductifact/internal/infrastructure/auth"
	"ductifact/internal/infrastructure/logging"
	"ductifact/internal/infrastructure/ratelimit"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
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

	// --- Client wiring ---
	clientRepo := persistence.NewPostgresClientRepository(db)

	// --- Project wiring ---
	projectRepo := persistence.NewPostgresProjectRepository(db)

	// --- Order wiring ---
	orderRepo := persistence.NewPostgresOrderRepository(db)

	// --- Piece Definition wiring ---
	pieceDefRepo := persistence.NewPostgresPieceDefinitionRepository(db)

	fileStorage, err := storage.NewMinIOStorage(
		cfg.MinIO.Endpoint,
		cfg.MinIO.AccessKey,
		cfg.MinIO.SecretKey,
		cfg.MinIO.Bucket,
		cfg.MinIO.UseSSL,
	)
	if err != nil {
		slog.Error("failed to connect to MinIO", "error", err)
		os.Exit(1)
	}

	imageProcessor := imageproc.NewImagingProcessor()

	// --- Piece wiring ---
	pieceRepo := persistence.NewPostgresPieceRepository(db)

	// --- Services (all repos must be created before services) ---
	userService := services.NewUserService(userRepo, clientRepo)
	clientService := services.NewClientService(clientRepo, userRepo, projectRepo)
	projectService := services.NewProjectService(projectRepo, clientRepo, orderRepo)
	orderService := services.NewOrderService(orderRepo, projectRepo, pieceRepo)
	pieceDefService := services.NewPieceDefinitionService(pieceDefRepo, fileStorage, imageProcessor, pieceRepo)
	pieceService := services.NewPieceService(pieceRepo, pieceDefRepo, orderRepo)

	// --- Email wiring ---
	emailSender := email.NewSMTPSender(
		cfg.SMTP.Host, cfg.SMTP.Port,
		cfg.SMTP.UseAuth,
		cfg.SMTP.Username, cfg.SMTP.Password,
		cfg.SMTP.From,
	)
	slog.Info("email sender ready", "host", cfg.SMTP.Host, "port", cfg.SMTP.Port)

	// --- Auth wiring ---
	tokenProvider := auth.NewJWTProvider(cfg.JWT)

	// --- Redis connection (primary backend for distributed state) ---
	var redisClient *redis.Client
	var redisChecker *persistence.RedisHealthChecker

	redisOpts := &redis.Options{
		Addr: cfg.Redis.Addr(),
		DB:   cfg.Redis.DB,
	}
	if cfg.Redis.UseAuth {
		redisOpts.Password = cfg.Redis.Password
	}
	redisClient = redis.NewClient(redisOpts)

	// Try to connect — if Redis is unavailable, degrade to in-memory adapters
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
	redisAvailable := redisClient.Ping(pingCtx).Err() == nil
	pingCancel()

	var blacklist ports.TokenBlacklist
	var loginThrottler ports.LoginThrottler
	var ipLimiter ports.RateLimiter
	var userLimiter ports.RateLimiter

	if redisAvailable {
		slog.Info("redis connected, using distributed adapters", "addr", cfg.Redis.Addr())
		redisChecker = persistence.NewRedisHealthChecker(redisClient)

		blacklist = auth.NewRedisBlacklist(redisClient)
		loginThrottler = ratelimit.NewRedisLoginThrottler(
			redisClient,
			cfg.LoginThrottle.MaxAttempts,
			cfg.LoginThrottle.Window,
			cfg.LoginThrottle.LockoutDuration,
		)
		ipLimiter = ratelimit.NewRedisRateLimiter(
			redisClient,
			cfg.RateLimit.IPMaxRequests,
			cfg.RateLimit.IPWindow,
		)
		userLimiter = ratelimit.NewRedisRateLimiter(
			redisClient,
			cfg.RateLimit.UserMaxRequests,
			cfg.RateLimit.UserWindow,
		)
	} else {
		slog.Error("redis unavailable, degrading to in-memory adapters", "addr", cfg.Redis.Addr())
		redisClient.Close()
		redisClient = nil

		memBlacklist := auth.NewMemoryBlacklist(5 * time.Minute)
		defer memBlacklist.Stop()
		blacklist = memBlacklist

		memThrottler := ratelimit.NewMemoryLoginThrottler(
			cfg.LoginThrottle.MaxAttempts,
			cfg.LoginThrottle.Window,
			cfg.LoginThrottle.LockoutDuration,
			1*time.Minute,
		)
		defer memThrottler.Stop()
		loginThrottler = memThrottler

		memIPLimiter := ratelimit.NewMemoryRateLimiter(
			cfg.RateLimit.IPMaxRequests,
			cfg.RateLimit.IPWindow,
			1*time.Minute,
		)
		defer memIPLimiter.Stop()
		ipLimiter = memIPLimiter

		memUserLimiter := ratelimit.NewMemoryRateLimiter(
			cfg.RateLimit.UserMaxRequests,
			cfg.RateLimit.UserWindow,
			1*time.Minute,
		)
		defer memUserLimiter.Stop()
		userLimiter = memUserLimiter
	}

	// --- One-time token repository ---
	oneTimeTokenRepo := persistence.NewPostgresOneTimeTokenRepository(db)

	authService := services.NewAuthService(
		userRepo,
		oneTimeTokenRepo,
		tokenProvider,
		blacklist,
		loginThrottler,
		emailSender,
		cfg.JWT.TokenDuration,
		cfg.JWT.RefreshTokenDuration,
		cfg.OneTimeTokens.EmailVerificationTTL,
		cfg.OneTimeTokens.PasswordResetTTL,
		cfg.OneTimeTokens.VerificationBaseURL,
	)

	// --- Health checker ---
	healthChecker := persistence.NewPostgresHealthChecker(db)

	// --- HTTP server ---
	router := httpAdapter.SetupRoutes(
		healthChecker,
		fileStorage,
		emailSender,
		redisChecker,
		userService,
		clientService,
		projectService,
		orderService,
		pieceDefService,
		pieceService,
		authService,
		tokenProvider,
		blacklist,
		ipLimiter,
		userLimiter,
		cfg.CORS,
		cfg.Log.Level,
	)

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

	// Close Redis connection
	if redisClient != nil {
		redisClient.Close()
	}

	slog.Info("server stopped gracefully")
}
