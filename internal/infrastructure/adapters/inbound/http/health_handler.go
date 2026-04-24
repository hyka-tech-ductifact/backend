package http

import (
	"net/http"
	"time"

	"ductifact/internal/application/ports"

	"github.com/gin-gonic/gin"
)

// HealthHandler provides liveness and readiness endpoints for the API.
//
// Kubernetes probes:
//   - GET /healthz  → liveness  (is the process alive?)
//   - GET /readyz   → readiness (can it serve traffic?)
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

// Healthz responds 200 if the process is alive. No dependency checks.
// Used as a Kubernetes liveness probe — a failure triggers a pod restart.
//
// Response 200:
//
//	{ "status": "alive" }
func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

// Readyz checks that the API and its dependencies are healthy.
// Used as a Kubernetes readiness probe — a failure removes the pod from the
// load balancer but does NOT restart it.
//
// Response 200 (ready):
//
//	{
//	  "status": "ready",
//	  "uptime": "2h35m10s",
//	  "database": "connected",
//	  "contract_version": "1.0.0"
//	}
//
// Response 503 (not ready):
//
//	{
//	  "status": "not_ready",
//	  "uptime": "2h35m10s",
//	  "database": "disconnected",
//	  "error": "connection refused",
//	  "contract_version": "1.0.0"
//	}
func (h *HealthHandler) Readyz(c *gin.Context) {
	uptime := time.Since(h.startTime).Round(time.Second).String()

	if err := h.healthChecker.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":           "not_ready",
			"uptime":           uptime,
			"database":         "disconnected",
			"error":            err.Error(),
			"contract_version": h.contractVersion,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           "ready",
		"uptime":           uptime,
		"database":         "connected",
		"contract_version": h.contractVersion,
	})
}
