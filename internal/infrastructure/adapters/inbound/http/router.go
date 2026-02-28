package http
package http

import (
	"event-service/internal/application/ports"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the HTTP router.
// It receives the inbound port (EventService interface), not a concrete implementation.
func SetupRoutes(eventService ports.EventService) *gin.Engine {
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

	return r
}
