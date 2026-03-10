package ports

import "context"

// HealthChecker is the outbound port for verifying infrastructure health.
// It is defined as an interface so the HTTP layer doesn't depend on
// a specific database library — the implementation lives in infrastructure.
type HealthChecker interface {
	Ping(ctx context.Context) error
}
