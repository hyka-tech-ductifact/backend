package http

import (
	"context"
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
	storageChecker  storagePinger
	emailChecker    emailPinger
	startTime       time.Time
	contractVersion string
	version         string
	commit          string
}

// storagePinger is a minimal interface for checking storage health.
// ports.FileStorage satisfies it (via its Ping method).
type storagePinger interface {
	Ping(ctx context.Context) error
}

// emailPinger is a minimal interface for checking email backend health.
// ports.EmailSender satisfies it (via its Ping method).
type emailPinger interface {
	Ping(ctx context.Context) error
}

// NewHealthHandler creates a new HealthHandler.
// Call this at application startup and pass the time the app started.
func NewHealthHandler(
	healthChecker ports.HealthChecker,
	storageChecker ports.FileStorage,
	emailChecker ports.EmailSender,
	startTime time.Time,
	contractVersion string,
	version string,
	commit string,
) *HealthHandler {
	return &HealthHandler{
		healthChecker:   healthChecker,
		storageChecker:  storageChecker,
		emailChecker:    emailChecker,
		startTime:       startTime,
		contractVersion: contractVersion,
		version:         version,
		commit:          commit,
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
//	  "storage": "connected",
//	  "email": "connected",
//	  "version": "v1.0.0",
//	  "commit": "abc1234",
//	  "contract_version": "1.0.0"
//	}
//
// Response 503 (not ready):
//
//	{
//	  "status": "not_ready",
//	  "uptime": "2h35m10s",
//	  "database": "disconnected",
//	  "storage": "connected",
//	  "email": "disconnected",
//	  "errors": ["database: connection refused", "email: smtp unreachable"],
//	  "version": "v1.0.0",
//	  "commit": "abc1234",
//	  "contract_version": "1.0.0"
//	}
//
// readyzResponse defines the JSON field order for the readiness endpoint.
type readyzResponse struct {
	Status          string   `json:"status"`
	Uptime          string   `json:"uptime"`
	Database        string   `json:"database"`
	Storage         string   `json:"storage"`
	Email           string   `json:"email"`
	Version         string   `json:"version"`
	Commit          string   `json:"commit"`
	ContractVersion string   `json:"contract_version"`
	Errors          []string `json:"errors,omitempty"`
}

func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx := c.Request.Context()
	uptime := time.Since(h.startTime).Round(time.Second).String()

	dbStatus := "connected"
	storageStatus := "connected"
	emailStatus := "connected"
	var errs []string

	if err := h.healthChecker.Ping(ctx); err != nil {
		dbStatus = "disconnected"
		errs = append(errs, "database: "+err.Error())
	}

	if err := h.storageChecker.Ping(ctx); err != nil {
		storageStatus = "disconnected"
		errs = append(errs, "storage: "+err.Error())
	}

	if err := h.emailChecker.Ping(ctx); err != nil {
		emailStatus = "disconnected"
		errs = append(errs, "email: "+err.Error())
	}

	status := "ready"
	code := http.StatusOK
	if len(errs) > 0 {
		status = "not_ready"
		code = http.StatusServiceUnavailable
	}

	body := readyzResponse{
		Status:          status,
		Uptime:          uptime,
		Database:        dbStatus,
		Storage:         storageStatus,
		Email:           emailStatus,
		Version:         h.version,
		Commit:          h.commit,
		ContractVersion: h.contractVersion,
	}
	if len(errs) > 0 {
		body.Errors = errs
	}

	c.JSON(code, body)
}
