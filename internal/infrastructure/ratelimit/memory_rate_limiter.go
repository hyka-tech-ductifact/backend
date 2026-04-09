package ratelimit

import (
	"sync"
	"time"
)

// entry tracks the request count for a single key within the current window.
type entry struct {
	count     int
	windowEnd time.Time
}

// MemoryRateLimiter implements ports.RateLimiter using an in-memory
// fixed-window counter. Each key (IP, user ID) gets up to `maxRequests`
// requests per `window`. When the window expires, the counter resets.
//
// This is suitable for single-instance deployments. For multi-instance
// deployments, replace with a Redis-backed implementation.
type MemoryRateLimiter struct {
	maxRequests int
	window      time.Duration

	mu      sync.Mutex
	entries map[string]*entry

	done chan struct{}
	now  func() time.Time // injectable clock for testing
}

// NewMemoryRateLimiter creates a rate limiter that allows maxRequests per window.
// It starts a background goroutine that cleans up expired entries
// every cleanupInterval. Call Stop() to terminate the goroutine.
func NewMemoryRateLimiter(maxRequests int, window, cleanupInterval time.Duration) *MemoryRateLimiter {
	return NewMemoryRateLimiterWithClock(maxRequests, window, cleanupInterval)
}

// NewMemoryRateLimiterWithClock is like NewMemoryRateLimiter but uses the real
// clock by default. Exported so tests can create instances without the cleanup
// goroutine (using the same constructor) while keeping the public API simple.
func NewMemoryRateLimiterWithClock(maxRequests int, window, cleanupInterval time.Duration) *MemoryRateLimiter {
	rl := &MemoryRateLimiter{
		maxRequests: maxRequests,
		window:      window,
		entries:     make(map[string]*entry),
		done:        make(chan struct{}),
		now:         time.Now,
	}

	go rl.cleanup(cleanupInterval)

	return rl
}

// Allow checks if the key has remaining requests in the current window.
// Returns true if the request is allowed, false if rate-limited.
func (rl *MemoryRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()

	e, exists := rl.entries[key]
	if !exists || now.After(e.windowEnd) {
		// New key or window expired → start a fresh window
		rl.entries[key] = &entry{
			count:     1,
			windowEnd: now.Add(rl.window),
		}
		return true
	}

	// Window still active — check the limit
	if e.count >= rl.maxRequests {
		return false
	}

	e.count++
	return true
}

// Stop terminates the background cleanup goroutine.
// Call this during graceful shutdown.
func (rl *MemoryRateLimiter) Stop() {
	close(rl.done)
}

// cleanup periodically removes expired entries to prevent memory leaks.
func (rl *MemoryRateLimiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := rl.now()
			for key, e := range rl.entries {
				if now.After(e.windowEnd) {
					delete(rl.entries, key)
				}
			}
			rl.mu.Unlock()
		case <-rl.done:
			return
		}
	}
}
