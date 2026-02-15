package http

import (
	"event-service/internal/application/usecases"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(eventUseCase *usecases.EventUseCase) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy !!!!"})
	})

	// Event routes
	eventHandler := NewEventHandler(eventUseCase)
	eventRoutes := r.Group("/events")
	{
		eventRoutes.POST("", eventHandler.CreateEvent)
		eventRoutes.GET("/:id", eventHandler.GetEvent)
	}

	return r
}
