package persistence

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// RedisHealthChecker implements ports.HealthChecker for Redis.
// Used specifically for the /readyz endpoint to verify Redis connectivity.
type RedisHealthChecker struct {
	client *redis.Client
}

// NewRedisHealthChecker creates a new RedisHealthChecker.
func NewRedisHealthChecker(client *redis.Client) *RedisHealthChecker {
	return &RedisHealthChecker{client: client}
}

// Ping verifies that Redis is reachable and responding.
func (h *RedisHealthChecker) Ping(ctx context.Context) error {
	return h.client.Ping(ctx).Err()
}
