package http

import (
	"ductifact/internal/application/ports"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the HTTP router.
func SetupRoutes(userService ports.UserService) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy !!!!"})
	})

	// User routes
	userHandler := NewUserHandler(userService)
	userRoutes := r.Group("/users")
	{
		userRoutes.POST("", userHandler.CreateUser)
		userRoutes.GET("/:id", userHandler.GetUser)
		userRoutes.PUT("/:id", userHandler.UpdateUser)
	}

	return r
}
