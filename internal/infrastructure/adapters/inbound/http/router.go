package http

import (
	"ductifact/internal/application/ports"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the HTTP router.
func SetupRoutes(userService ports.UserService, clientService ports.ClientService) *gin.Engine {
	r := gin.Default()

	// API v1
	v1 := r.Group("/api/v1")

	v1.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy !!!!"})
	})

	// User routes
	userHandler := NewUserHandler(userService)
	userRoutes := v1.Group("/users")
	{
		userRoutes.GET("/:user_id", userHandler.GetUser)
		userRoutes.PUT("/:user_id", userHandler.UpdateUser)
	}

	// Client routes (nested under users — a client belongs to a user)
	clientHandler := NewClientHandler(clientService)
	clientRoutes := userRoutes.Group("/:user_id/clients")
	{
		clientRoutes.POST("", clientHandler.CreateClient)
		clientRoutes.GET("", clientHandler.ListClients)
		clientRoutes.GET("/:client_id", clientHandler.GetClient)
		clientRoutes.PUT("/:client_id", clientHandler.UpdateClient)
		clientRoutes.DELETE("/:client_id", clientHandler.DeleteClient)
	}

	return r
}
