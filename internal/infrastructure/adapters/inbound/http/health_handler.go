package http

import (
	"context"
	"log/slog"
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
	redisChecker    redisPinger
	startTime       time.Time
	contractVersion string
	version         string
	commit          string
	logLevel        string
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

// redisPinger is a minimal interface for checking Redis health.
// Can be nil when running in memory-only mode.
type redisPinger interface {
	Ping(ctx context.Context) error
}

// NewHealthHandler creates a new HealthHandler.
// Call this at application startup and pass the time the app started.
func NewHealthHandler(
	healthChecker ports.HealthChecker,
	storageChecker ports.FileStorage,
	emailChecker ports.EmailSender,
	redisChecker redisPinger,
	startTime time.Time,
	contractVersion string,
	version string,
	commit string,
	logLevel string,
) *HealthHandler {
	return &HealthHandler{
		healthChecker:   healthChecker,
		storageChecker:  storageChecker,
		emailChecker:    emailChecker,
		redisChecker:    redisChecker,
		startTime:       startTime,
		contractVersion: contractVersion,
		version:         version,
		commit:          commit,
		logLevel:        logLevel,
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

// Readyz checks that the API and its critical dependencies are healthy.
// Used as a Kubernetes readiness probe — a failure removes the pod from the
// load balancer but does NOT restart it.
//
// Critical dependencies (cause 503): database, storage.
// Non-critical dependencies (cause "degraded" status but still 200): email.
//
// In production, error details are not exposed in the response; they are
// logged internally instead.
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
// Response 200 (degraded — non-critical dependency down):
//
//	{
//	  "status": "degraded",
//	  "uptime": "2h35m10s",
//	  "database": "connected",
//	  "storage": "connected",
//	  "email": "unavailable",
//	  "warnings": ["email: unavailable"],
//	  "version": "v1.0.0",
//	  "commit": "abc1234",
//	  "contract_version": "1.0.0"
//	}
//
// Response 503 (not ready — critical dependency down):
//
//	{
//	  "status": "not_ready",
//	  "uptime": "2h35m10s",
//	  "database": "disconnected",
//	  "storage": "connected",
//	  "email": "connected",
//	  "errors": ["database: unavailable"],
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
	Redis           string   `json:"redis"`
	Email           string   `json:"email"`
	Version         string   `json:"version"`
	Commit          string   `json:"commit"`
	ContractVersion string   `json:"contract_version"`
	Errors          []string `json:"errors,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
}

func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx := c.Request.Context()
	uptime := time.Since(h.startTime).Round(time.Second).String()

	dbStatus := "connected"
	storageStatus := "connected"
	redisStatus := "connected"
	emailStatus := "connected"
	var errs []string
	var warnings []string

	// Critical checks — failure causes 503
	if err := h.healthChecker.Ping(ctx); err != nil {
		dbStatus = "disconnected"
		if h.logLevel == "debug" {
			errs = append(errs, "database: "+err.Error())
		} else {
			slog.Error("readyz: database ping failed", "error", err.Error())
			errs = append(errs, "database: unavailable")
		}
	}

	if err := h.storageChecker.Ping(ctx); err != nil {
		storageStatus = "disconnected"
		if h.logLevel == "debug" {
			errs = append(errs, "storage: "+err.Error())
		} else {
			slog.Error("readyz: storage ping failed", "error", err.Error())
			errs = append(errs, "storage: unavailable")
		}
	}

	// Non-critical check — failure causes "degraded" but NOT 503
	if h.redisChecker != nil {
		if err := h.redisChecker.Ping(ctx); err != nil {
			redisStatus = "unavailable"
			if h.logLevel == "debug" {
				warnings = append(warnings, "redis: "+err.Error())
			} else {
				slog.Warn("readyz: redis ping failed", "error", err.Error())
				warnings = append(warnings, "redis: unavailable")
			}
		}
	} else {
		redisStatus = "not_configured"
	}

	// Non-critical check — failure causes "degraded" but NOT 503
	if err := h.emailChecker.Ping(ctx); err != nil {
		emailStatus = "unavailable"
		if h.logLevel == "debug" {
			warnings = append(warnings, "email: "+err.Error())
		} else {
			slog.Warn("readyz: email ping failed", "error", err.Error())
			warnings = append(warnings, "email: unavailable")
		}
	}

	status := "ready"
	code := http.StatusOK
	if len(errs) > 0 {
		status = "not_ready"
		code = http.StatusServiceUnavailable
	} else if len(warnings) > 0 {
		status = "degraded"
	}

	body := readyzResponse{
		Status:          status,
		Uptime:          uptime,
		Database:        dbStatus,
		Storage:         storageStatus,
		Redis:           redisStatus,
		Email:           emailStatus,
		Version:         h.version,
		Commit:          h.commit,
		ContractVersion: h.contractVersion,
	}
	if len(errs) > 0 {
		body.Errors = errs
	}
	if len(warnings) > 0 {
		body.Warnings = warnings
	}

	c.JSON(code, body)
}
