package http

import (
	"net/http"
	"time"

	"ductifact/internal/application/ports"
	"ductifact/internal/application/services"
	"ductifact/internal/application/usecases"
	"ductifact/internal/config"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"
	"ductifact/internal/domain/valueobjects"
	"ductifact/internal/infrastructure/adapters/inbound/http/helpers"
	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRoutes configures the HTTP router with public and protected route groups.
// Protected routes require a valid JWT in the Authorization header.
func SetupRoutes(
	healthChecker ports.HealthChecker,
	userService usecases.UserService,
	clientService usecases.ClientService,
	projectService usecases.ProjectService,
	orderService usecases.OrderService,
	pieceDefService usecases.PieceDefinitionService,
	pieceService usecases.PieceService,
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
	helpers.RegisterDomainError(services.ErrAccountLocked, http.StatusTooManyRequests, "account temporarily locked, please try again later")
	helpers.RegisterDomainError(repositories.ErrClientNotFound, http.StatusNotFound, "client not found")
	helpers.RegisterDomainError(repositories.ErrClientNotOwned, http.StatusNotFound, "client not found")
	helpers.RegisterDomainError(repositories.ErrProjectNotFound, http.StatusNotFound, "project not found")
	helpers.RegisterDomainError(repositories.ErrProjectNotOwned, http.StatusNotFound, "project not found")
	helpers.RegisterDomainError(entities.ErrEmptyUserName, http.StatusBadRequest, "user name cannot be empty")
	helpers.RegisterDomainError(entities.ErrEmptyClientName, http.StatusBadRequest, "client name cannot be empty")
	helpers.RegisterDomainError(entities.ErrEmptyProjectName, http.StatusBadRequest, "project name cannot be empty")
	helpers.RegisterDomainError(repositories.ErrOrderNotFound, http.StatusNotFound, "order not found")
	helpers.RegisterDomainError(repositories.ErrOrderNotOwned, http.StatusNotFound, "order not found")
	helpers.RegisterDomainError(entities.ErrEmptyOrderTitle, http.StatusBadRequest, "order title cannot be empty")
	helpers.RegisterDomainError(entities.ErrInvalidOrderStatus, http.StatusBadRequest, "order status must be 'pending' or 'completed'")
	helpers.RegisterDomainError(repositories.ErrPieceDefNotFound, http.StatusNotFound, "piece definition not found")
	helpers.RegisterDomainError(entities.ErrEmptyPieceDefName, http.StatusBadRequest, "piece definition name cannot be empty")
	helpers.RegisterDomainError(entities.ErrTooManyDimensionFields, http.StatusBadRequest, "piece definition cannot have more than 10 dimension fields")
	helpers.RegisterDomainError(entities.ErrNoDimensionFields, http.StatusBadRequest, "piece definition must have at least one dimension field")
	helpers.RegisterDomainError(entities.ErrDuplicateDimensionLabel, http.StatusBadRequest, "dimension labels must be unique")
	helpers.RegisterDomainError(entities.ErrEmptyDimensionLabel, http.StatusBadRequest, "dimension label cannot be empty")
	helpers.RegisterDomainError(services.ErrPieceDefPredefined, http.StatusForbidden, "predefined piece definitions cannot be modified")
	helpers.RegisterDomainError(repositories.ErrPieceDefNotOwned, http.StatusNotFound, "piece definition not found")
	helpers.RegisterDomainError(repositories.ErrPieceNotFound, http.StatusNotFound, "piece not found")
	helpers.RegisterDomainError(repositories.ErrPieceNotOwned, http.StatusNotFound, "piece not found")
	helpers.RegisterDomainError(entities.ErrEmptyPieceTitle, http.StatusBadRequest, "piece title cannot be empty")
	helpers.RegisterDomainError(entities.ErrInvalidPieceQuantity, http.StatusBadRequest, "piece quantity must be at least 1")
	helpers.RegisterDomainError(entities.ErrMissingDimensions, http.StatusBadRequest, "missing required dimensions")
	helpers.RegisterDomainError(entities.ErrUnexpectedDimensions, http.StatusBadRequest, "unexpected dimensions")
	helpers.RegisterDomainError(entities.ErrInvalidDimensionValues, http.StatusBadRequest, "dimension values must be positive")
	helpers.RegisterDomainError(valueobjects.ErrPasswordTooShort, http.StatusBadRequest, "password must be at least 8 characters")
	helpers.RegisterDomainError(valueobjects.ErrPasswordTooLong, http.StatusBadRequest, "password must not exceed 72 characters")
	helpers.RegisterDomainError(valueobjects.ErrPasswordEmpty, http.StatusBadRequest, "password cannot be empty")
	helpers.RegisterDomainError(valueobjects.ErrInvalidEmail, http.StatusBadRequest, "invalid email format")

	// --- Reject unknown JSON fields (RFC 7231 §6.5.1) ---
	// Makes ShouldBindJSON return 400 for bodies with extra properties.
	binding.EnableDecoderDisallowUnknownFields = true

	// --- Create router WITHOUT default middlewares ---
	r := gin.New()

	// Return 405 Method Not Allowed (with Allow header) instead of 404
	// for routes that exist but don't support the requested HTTP method.
	r.HandleMethodNotAllowed = true

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
	r.Use(middleware.CORSMiddleware(corsCfg.AllowedOrigins))

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

	// Client routes — scoped by JWT (userID from token)
	clientHandler := NewClientHandler(clientService)
	clientRoutes := protected.Group("/clients")
	{
		clientRoutes.POST("", clientHandler.CreateClient)
		clientRoutes.GET("", clientHandler.ListClients)
		clientRoutes.GET("/:client_id", clientHandler.GetClient)
		clientRoutes.PUT("/:client_id", clientHandler.UpdateClient)
		clientRoutes.DELETE("/:client_id", clientHandler.DeleteClient)
	}

	// Project routes — collection nested under client, item flat
	projectHandler := NewProjectHandler(projectService)
	protected.POST("/clients/:client_id/projects", projectHandler.CreateProject)
	protected.GET("/clients/:client_id/projects", projectHandler.ListProjects)
	projectRoutes := protected.Group("/projects")
	{
		projectRoutes.GET("/:project_id", projectHandler.GetProject)
		projectRoutes.PUT("/:project_id", projectHandler.UpdateProject)
		projectRoutes.DELETE("/:project_id", projectHandler.DeleteProject)
	}

	// Order routes — collection nested under project, item flat
	orderHandler := NewOrderHandler(orderService)
	protected.POST("/projects/:project_id/orders", orderHandler.CreateOrder)
	protected.GET("/projects/:project_id/orders", orderHandler.ListOrders)
	orderRoutes := protected.Group("/orders")
	{
		orderRoutes.GET("/:order_id", orderHandler.GetOrder)
		orderRoutes.PUT("/:order_id", orderHandler.UpdateOrder)
		orderRoutes.DELETE("/:order_id", orderHandler.DeleteOrder)
	}

	// Piece Definition routes — flat, user-scoped (predefined + custom)
	pieceDefHandler := NewPieceDefinitionHandler(pieceDefService)
	pieceDefRoutes := protected.Group("/piece-definitions")
	{
		pieceDefRoutes.POST("", pieceDefHandler.CreatePieceDefinition)
		pieceDefRoutes.GET("", pieceDefHandler.ListPieceDefinitions)
		pieceDefRoutes.GET("/:piece_definition_id", pieceDefHandler.GetPieceDefinition)
		pieceDefRoutes.PUT("/:piece_definition_id", pieceDefHandler.UpdatePieceDefinition)
		pieceDefRoutes.DELETE("/:piece_definition_id", pieceDefHandler.DeletePieceDefinition)
	}

	// Piece routes — always nested under order (weak entity)
	pieceHandler := NewPieceHandler(pieceService)
	pieceRoutes := protected.Group("/orders/:order_id/pieces")
	{
		pieceRoutes.POST("", pieceHandler.CreatePiece)
		pieceRoutes.GET("", pieceHandler.ListPieces)
		pieceRoutes.GET("/:piece_id", pieceHandler.GetPiece)
		pieceRoutes.PUT("/:piece_id", pieceHandler.UpdatePiece)
		pieceRoutes.DELETE("/:piece_id", pieceHandler.DeletePiece)
	}

	return r
}
