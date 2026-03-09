package http

import (
	"net/http"
	"os"
	"strings"
	"time"

	"ductifact/internal/application/ports"
	"ductifact/internal/application/services"
	"ductifact/internal/application/usecases"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/valueobjects"
	"ductifact/internal/infrastructure/adapters/inbound/http/helpers"
	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the HTTP router with public and protected route groups.
// Protected routes require a valid JWT in the Authorization header.
func SetupRoutes(
	userService usecases.UserService,
	clientService usecases.ClientService,
	authService usecases.AuthService,
	tokenProvider ports.TokenProvider,
) *gin.Engine {
	// --- Register domain error → HTTP status mappings ---
	helpers.RegisterDomainError(services.ErrUserNotFound, http.StatusNotFound, "user not found")
	helpers.RegisterDomainError(services.ErrEmailAlreadyInUse, http.StatusConflict, "email already in use")
	helpers.RegisterDomainError(services.ErrEmailTaken, http.StatusConflict, "email already registered")
	helpers.RegisterDomainError(services.ErrInvalidCredentials, http.StatusUnauthorized, "invalid email or password")
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

	// 4. CORS: fourth, before any business logic
	// CORS_ORIGINS is a comma-separated list of allowed origins (e.g. "http://localhost:3000,http://localhost:5173")
	allowedOrigins := strings.Split(os.Getenv("CORS_ORIGINS"), ",")
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	v1 := r.Group("/api/v1")

	// --- Public routes (no auth required) ---

	v1.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy !!!!"})
	})

	// Auth routes
	authHandler := NewAuthHandler(authService)
	authRoutes := v1.Group("/auth")
	{
		authRoutes.POST("/register", authHandler.Register)
		authRoutes.POST("/login", authHandler.Login)
	}

	// --- Protected routes (auth required) ---

	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(tokenProvider))

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
