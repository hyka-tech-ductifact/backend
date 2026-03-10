package http

import (
	"net/http"
	"time"

	"ductifact/internal/application/ports"

	"github.com/gin-gonic/gin"
)

// HealthHandler provides health check endpoints for the API.
type HealthHandler struct {
	healthChecker ports.HealthChecker
	startTime     time.Time
}

// NewHealthHandler creates a new HealthHandler.
// Call this at application startup and pass the time the app started.
func NewHealthHandler(healthChecker ports.HealthChecker, startTime time.Time) *HealthHandler {
	return &HealthHandler{
		healthChecker: healthChecker,
		startTime:     startTime,
	}
}

// Check verifies that the API and its dependencies are healthy.
//
// Response 200 (healthy):
//
//	{
//	  "status": "healthy",
//	  "uptime": "2h35m10s",
//	  "database": "connected"
//	}
//
// Response 503 (unhealthy):
//
//	{
//	  "status": "unhealthy",
//	  "uptime": "2h35m10s",
//	  "database": "disconnected",
//	  "error": "connection refused"
//	}
func (h *HealthHandler) Check(c *gin.Context) {
	uptime := time.Since(h.startTime).Round(time.Second).String()

	if err := h.healthChecker.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unhealthy",
			"uptime":   uptime,
			"database": "disconnected",
			"error":    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"uptime":   uptime,
		"database": "connected",
	})
}
