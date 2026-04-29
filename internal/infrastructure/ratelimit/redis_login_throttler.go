package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	throttleAttemptsPrefix = "login_attempts:"
	throttleLockPrefix     = "login_lock:"
)

// RedisLoginThrottler implements ports.LoginThrottler using Redis.
// It tracks failed login attempts per key (email) and temporarily blocks
// the account after maxAttempts failures within a window.
//
// Uses two keys per email:
//   - login_attempts:<email> — counter with TTL = window
//   - login_lock:<email>     — presence key with TTL = lockout duration
//
// This is suitable for multi-instance deployments where all instances
// need to share the same throttle state.
type RedisLoginThrottler struct {
	client          *redis.Client
	maxAttempts     int
	window          time.Duration
	lockoutDuration time.Duration
}

// NewRedisLoginThrottler creates a login throttler backed by Redis.
// No background cleanup goroutine is needed — Redis TTL handles expiry.
func NewRedisLoginThrottler(
	client *redis.Client,
	maxAttempts int,
	window, lockoutDuration time.Duration,
) *RedisLoginThrottler {
	return &RedisLoginThrottler{
		client:          client,
		maxAttempts:     maxAttempts,
		window:          window,
		lockoutDuration: lockoutDuration,
	}
}

// IsBlocked returns true if the given key (email) has exceeded
// the maximum number of failed login attempts and is temporarily locked.
func (lt *RedisLoginThrottler) IsBlocked(key string) bool {
	ctx := context.Background()
	result, err := lt.client.Exists(ctx, throttleLockPrefix+key).Result()
	if err != nil {
		// On Redis error, do NOT block the user (fail-open for availability)
		return false
	}
	return result > 0
}

// RecordFailure increments the failed attempt counter for the given key.
// After reaching the configured threshold, the key is locked out.
func (lt *RedisLoginThrottler) RecordFailure(key string) {
	ctx := context.Background()
	attemptsKey := throttleAttemptsPrefix + key

	// Increment the failure counter (creates at 1 if missing)
	count, err := lt.client.Incr(ctx, attemptsKey).Result()
	if err != nil {
		return
	}

	// Set expiry on first failure (start the window)
	if count == 1 {
		lt.client.Expire(ctx, attemptsKey, lt.window)
	}

	// If threshold reached, lock the account
	if count >= int64(lt.maxAttempts) {
		lt.client.Set(ctx, throttleLockPrefix+key, "1", lt.lockoutDuration)
		// Clear the attempts counter — the lock key now controls access
		lt.client.Del(ctx, attemptsKey)
	}
}

// Reset clears the failure counter and lock for the given key.
// Call this after a successful login.
func (lt *RedisLoginThrottler) Reset(key string) {
	ctx := context.Background()
	lt.client.Del(ctx, throttleAttemptsPrefix+key, throttleLockPrefix+key)
}

// Stop is a no-op for Redis — no background goroutine to terminate.
func (lt *RedisLoginThrottler) Stop() {}
