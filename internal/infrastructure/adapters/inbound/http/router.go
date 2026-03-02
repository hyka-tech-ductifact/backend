package http

import (
	"event-service/internal/application/ports"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the HTTP router.
// It receives the inbound port (EventService interface), not a concrete implementation.
func SetupRoutes(eventService ports.EventService, userService ports.UserService) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy !!!!"})
	})

	// Event routes
	eventHandler := NewEventHandler(eventService)
	eventRoutes := r.Group("/events")
	{
		eventRoutes.POST("", eventHandler.CreateEvent)
		eventRoutes.GET("/:id", eventHandler.GetEvent)
	}

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
