package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const rateLimitPrefix = "rate:"

// RedisRateLimiter implements ports.RateLimiter using Redis.
// It uses a fixed-window counter per key with automatic expiry.
//
// This is suitable for multi-instance deployments where all instances
// need to share the same rate limit state.
type RedisRateLimiter struct {
	client      *redis.Client
	maxRequests int
	window      time.Duration
}

// NewRedisRateLimiter creates a rate limiter that allows maxRequests per window.
// No background cleanup goroutine is needed — Redis TTL handles expiry.
func NewRedisRateLimiter(client *redis.Client, maxRequests int, window time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client:      client,
		maxRequests: maxRequests,
		window:      window,
	}
}

// Allow checks if the key has remaining requests in the current window.
// Returns true if the request is allowed, false if rate-limited.
//
// Uses INCR + EXPIRE in Redis:
//   - INCR atomically increments the counter (creates it at 1 if missing)
//   - EXPIRE sets the TTL only on the first request of a new window
//   - If counter > maxRequests → reject
func (rl *RedisRateLimiter) Allow(key string) bool {
	ctx := context.Background()
	redisKey := rateLimitPrefix + key

	// INCR is atomic — if the key doesn't exist, Redis creates it with value 1
	count, err := rl.client.Incr(ctx, redisKey).Result()
	if err != nil {
		// On Redis error, allow the request (fail-open for availability)
		return true
	}

	// First request in this window — set the expiry
	if count == 1 {
		rl.client.Expire(ctx, redisKey, rl.window)
	}

	return count <= int64(rl.maxRequests)
}

// Stop is a no-op for Redis — no background goroutine to terminate.
func (rl *RedisRateLimiter) Stop() {}
