package auth

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const blacklistPrefix = "blacklist:"

// RedisBlacklist implements ports.TokenBlacklist using Redis.
// Tokens are stored with a TTL matching their remaining JWT lifetime,
// so Redis automatically removes them when they would have expired anyway.
//
// This is suitable for multi-instance deployments where all instances
// need to share the same blacklist state.
type RedisBlacklist struct {
	client *redis.Client
}

// NewRedisBlacklist creates a token blacklist backed by Redis.
// No background cleanup goroutine is needed — Redis TTL handles expiry.
func NewRedisBlacklist(client *redis.Client) *RedisBlacklist {
	return &RedisBlacklist{client: client}
}

// Add marks a token as revoked. It will be automatically removed
// by Redis after the given duration (matching the token's remaining lifetime).
func (b *RedisBlacklist) Add(token string, expiry time.Duration) {
	ctx := context.Background()
	b.client.Set(ctx, blacklistPrefix+token, "1", expiry)
}

// IsBlacklisted returns true if the token has been revoked.
func (b *RedisBlacklist) IsBlacklisted(token string) bool {
	ctx := context.Background()
	result, err := b.client.Exists(ctx, blacklistPrefix+token).Result()
	if err != nil {
		// On Redis error, assume NOT blacklisted to avoid blocking all users.
		// This is a trade-off: availability over strict security.
		// The graceful degradation (fallback to memory) in main.go handles
		// the case where Redis is completely unavailable at startup.
		return false
	}
	return result > 0
}

// Stop is a no-op for Redis — no background goroutine to terminate.
func (b *RedisBlacklist) Stop() {}
