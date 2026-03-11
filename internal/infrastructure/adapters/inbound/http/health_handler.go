package http

import (
	"net/http"
	"time"

	"ductifact/internal/application/ports"

	"github.com/gin-gonic/gin"
)

// HealthHandler provides health check endpoints for the API.
type HealthHandler struct {
	healthChecker   ports.HealthChecker
	startTime       time.Time
	contractVersion string
}

// NewHealthHandler creates a new HealthHandler.
// Call this at application startup and pass the time the app started.
// contractVersion is the semantic version of the API contract (e.g. "1.0.0").
func NewHealthHandler(healthChecker ports.HealthChecker, startTime time.Time, contractVersion string) *HealthHandler {
	return &HealthHandler{
		healthChecker:   healthChecker,
		startTime:       startTime,
		contractVersion: contractVersion,
	}
}

// Check verifies that the API and its dependencies are healthy.
//
// Response 200 (healthy):
//
//	{
//	  "status": "healthy",
//	  "uptime": "2h35m10s",
//	  "database": "connected",
//	  "contract_version": "1.0.0"
//	}
//
// Response 503 (unhealthy):
//
//	{
//	  "status": "unhealthy",
//	  "uptime": "2h35m10s",
//	  "database": "disconnected",
//	  "error": "connection refused",
//	  "contract_version": "1.0.0"
//	}
func (h *HealthHandler) Check(c *gin.Context) {
	uptime := time.Since(h.startTime).Round(time.Second).String()

	if err := h.healthChecker.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":           "unhealthy",
			"uptime":           uptime,
			"database":         "disconnected",
			"error":            err.Error(),
			"contract_version": h.contractVersion,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           "healthy",
		"uptime":           uptime,
		"database":         "connected",
		"contract_version": h.contractVersion,
	})
}
