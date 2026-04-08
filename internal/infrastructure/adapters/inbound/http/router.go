package http

import (
	"net/http"
	"time"

	"ductifact/internal/application/ports"
	"ductifact/internal/application/services"
	"ductifact/internal/application/usecases"
	"ductifact/internal/config"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/valueobjects"
	"ductifact/internal/infrastructure/adapters/inbound/http/helpers"
	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRoutes configures the HTTP router with public and protected route groups.
// Protected routes require a valid JWT in the Authorization header.
func SetupRoutes(
	healthChecker ports.HealthChecker,
	userService usecases.UserService,
	clientService usecases.ClientService,
	authService usecases.AuthService,
	tokenProvider ports.TokenProvider,
	blacklist ports.TokenBlacklist,
	ipLimiter ports.RateLimiter,
	userLimiter ports.RateLimiter,
	corsCfg config.CORS,
) *gin.Engine {
	// --- Register domain error → HTTP status mappings ---
	helpers.RegisterDomainError(services.ErrUserNotFound, http.StatusNotFound, "user not found")
	helpers.RegisterDomainError(services.ErrEmailAlreadyInUse, http.StatusConflict, "email already in use")
	helpers.RegisterDomainError(services.ErrInvalidCredentials, http.StatusUnauthorized, "invalid email or password")
	helpers.RegisterDomainError(services.ErrInvalidRefreshToken, http.StatusUnauthorized, "invalid or expired refresh token")
	helpers.RegisterDomainError(services.ErrClientNotFound, http.StatusNotFound, "client not found")
	helpers.RegisterDomainError(services.ErrClientNotOwned, http.StatusForbidden, "client does not belong to this user")
	helpers.RegisterDomainError(entities.ErrEmptyUserName, http.StatusBadRequest, "user name cannot be empty")
	helpers.RegisterDomainError(entities.ErrEmptyClientName, http.StatusBadRequest, "client name cannot be empty")
	helpers.RegisterDomainError(valueobjects.ErrPasswordTooShort, http.StatusBadRequest, "password must be at least 8 characters")
	helpers.RegisterDomainError(valueobjects.ErrPasswordEmpty, http.StatusBadRequest, "password cannot be empty")
	helpers.RegisterDomainError(valueobjects.ErrInvalidEmail, http.StatusBadRequest, "invalid email format")

	// --- Create router WITHOUT default middlewares ---
	r := gin.New()

	// --- Register middlewares in correct order ---
	// 1. Request ID: first, so all logs include the ID
	r.Use(middleware.RequestIDMiddleware())

	// 2. Logger: second, to log each request with the ID
	r.Use(middleware.LoggerMiddleware())

	// 3. Recovery: third, to catch panics from anything below
	r.Use(middleware.RecoveryMiddleware())

	// 4. Security headers: fourth, so ALL responses include them (even errors)
	r.Use(middleware.SecurityHeadersMiddleware())

	// 5. Metrics: fifth, to record request count and latency
	r.Use(middleware.MetricsMiddleware())

	// 6. CORS: sixth, before any business logic
	r.Use(cors.New(cors.Config{
		AllowOrigins:     corsCfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 7. Rate limit by IP: seventh, applied globally to all routes
	r.Use(middleware.IPRateLimitMiddleware(ipLimiter))

	// --- Infrastructure routes (unversioned) ---

	healthHandler := NewHealthHandler(healthChecker, time.Now(), config.ContractVersion)
	r.GET("/health", healthHandler.Check)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	docsHandler := NewDocsHandler()
	r.GET("/docs", docsHandler.UI)
	r.GET("/docs/openapi.yaml", docsHandler.Spec)

	// --- Versioned API routes ---

	v1 := r.Group("/v1")

	// Auth routes
	authHandler := NewAuthHandler(authService)
	authRoutes := v1.Group("/auth")
	{
		authRoutes.POST("/register", authHandler.Register)
		authRoutes.POST("/login", authHandler.Login)
		authRoutes.POST("/refresh", authHandler.Refresh)
	}

	// --- Protected routes (auth required) ---

	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(tokenProvider, blacklist))
	protected.Use(middleware.UserRateLimitMiddleware(userLimiter))

	// Auth routes that require authentication
	protectedAuth := protected.Group("/auth")
	{
		protectedAuth.POST("/logout", authHandler.Logout)
	}

	// User routes — userID comes from the JWT token, not the URL
	userHandler := NewUserHandler(userService)
	userRoutes := protected.Group("/users")
	{
		userRoutes.GET("/me", userHandler.GetMe)
		userRoutes.PUT("/me", userHandler.UpdateMe)
	}

	// Client routes — always "my" clients (userID from token)
	clientHandler := NewClientHandler(clientService)
	clientRoutes := protected.Group("/users/me/clients")
	{
		clientRoutes.POST("", clientHandler.CreateClient)
		clientRoutes.GET("", clientHandler.ListClients)
		clientRoutes.GET("/:client_id", clientHandler.GetClient)
		clientRoutes.PUT("/:client_id", clientHandler.UpdateClient)
		clientRoutes.DELETE("/:client_id", clientHandler.DeleteClient)
	}

	return r
}
