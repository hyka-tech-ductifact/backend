package auth

import (
	"sync"
	"time"
)

// MemoryBlacklist implements ports.TokenBlacklist using an in-memory store.
// Tokens are stored with their expiry time and cleaned up periodically.
//
// This is suitable for single-instance deployments. For multi-instance
// deployments, replace with a Redis-backed implementation.
type MemoryBlacklist struct {
	tokens sync.Map // map[string]time.Time (token → expiry)
	done   chan struct{}
}

// NewMemoryBlacklist creates a new in-memory token blacklist.
// It starts a background goroutine that cleans up expired entries
// every cleanupInterval. Call Stop() to terminate the goroutine.
func NewMemoryBlacklist(cleanupInterval time.Duration) *MemoryBlacklist {
	b := &MemoryBlacklist{
		done: make(chan struct{}),
	}

	go b.cleanup(cleanupInterval)

	return b
}

// Add marks a token as revoked until it naturally expires.
func (b *MemoryBlacklist) Add(token string, expiry time.Duration) {
	b.tokens.Store(token, time.Now().Add(expiry))
}

// IsBlacklisted returns true if the token has been revoked and hasn't expired yet.
func (b *MemoryBlacklist) IsBlacklisted(token string) bool {
	val, ok := b.tokens.Load(token)
	if !ok {
		return false
	}

	expiresAt := val.(time.Time)
	if time.Now().After(expiresAt) {
		// Token has naturally expired — remove it from the blacklist
		b.tokens.Delete(token)
		return false
	}

	return true
}

// Stop terminates the background cleanup goroutine.
// Call this during graceful shutdown.
func (b *MemoryBlacklist) Stop() {
	close(b.done)
}

// cleanup periodically removes expired entries from the blacklist.
func (b *MemoryBlacklist) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			b.tokens.Range(func(key, value any) bool {
				if now.After(value.(time.Time)) {
					b.tokens.Delete(key)
				}
				return true
			})
		case <-b.done:
			return
		}
	}
}
