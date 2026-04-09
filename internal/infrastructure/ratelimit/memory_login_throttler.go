package ratelimit

import (
	"sync"
	"time"
)

// throttleEntry tracks the failed login attempts for a single key (email).
type throttleEntry struct {
	failures    int
	lockedUntil time.Time
}

// MemoryLoginThrottler implements ports.LoginThrottler using an in-memory store.
// It tracks failed login attempts per key (email) and temporarily blocks
// the account after maxAttempts failures within a window.
//
// After being blocked, the account is locked for lockoutDuration.
// A successful login (Reset) clears the failure counter immediately.
//
// This is suitable for single-instance deployments. For multi-instance
// deployments, replace with a Redis-backed implementation.
type MemoryLoginThrottler struct {
	maxAttempts     int
	window          time.Duration
	lockoutDuration time.Duration

	mu      sync.Mutex
	entries map[string]*throttleEntry

	done chan struct{}
	now  func() time.Time // injectable clock for testing
}

// NewMemoryLoginThrottler creates a login throttler that allows maxAttempts
// failed logins per window before locking the account for lockoutDuration.
// It starts a background goroutine that cleans up expired entries.
func NewMemoryLoginThrottler(maxAttempts int, window, lockoutDuration, cleanupInterval time.Duration) *MemoryLoginThrottler {
	lt := &MemoryLoginThrottler{
		maxAttempts:     maxAttempts,
		window:          window,
		lockoutDuration: lockoutDuration,
		entries:         make(map[string]*throttleEntry),
		done:            make(chan struct{}),
		now:             time.Now,
	}

	go lt.cleanup(cleanupInterval)

	return lt
}

// IsBlocked returns true if the key is currently locked out due to
// too many failed login attempts.
func (lt *MemoryLoginThrottler) IsBlocked(key string) bool {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	e, exists := lt.entries[key]
	if !exists {
		return false
	}

	now := lt.now()

	// If there's an active lockout, check if it has expired
	if !e.lockedUntil.IsZero() && now.Before(e.lockedUntil) {
		return true
	}

	return false
}

// RecordFailure increments the failure counter for the given key.
// If the counter reaches maxAttempts, the key is locked out.
func (lt *MemoryLoginThrottler) RecordFailure(key string) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	e, exists := lt.entries[key]
	if !exists {
		lt.entries[key] = &throttleEntry{failures: 1}
		return
	}

	now := lt.now()

	// If lockout expired, reset and count this as the first new failure
	if !e.lockedUntil.IsZero() && now.After(e.lockedUntil) {
		e.failures = 1
		e.lockedUntil = time.Time{}
		return
	}

	e.failures++

	// If threshold reached, lock the account
	if e.failures >= lt.maxAttempts {
		e.lockedUntil = now.Add(lt.lockoutDuration)
	}
}

// Reset clears the failure counter for the given key.
// Call this after a successful login.
func (lt *MemoryLoginThrottler) Reset(key string) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	delete(lt.entries, key)
}

// Stop terminates the background cleanup goroutine.
func (lt *MemoryLoginThrottler) Stop() {
	close(lt.done)
}

// cleanup periodically removes expired entries to prevent memory leaks.
func (lt *MemoryLoginThrottler) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lt.mu.Lock()
			now := lt.now()
			for key, e := range lt.entries {
				// Remove entries whose lockout has expired
				if !e.lockedUntil.IsZero() && now.After(e.lockedUntil) {
					delete(lt.entries, key)
				}
			}
			lt.mu.Unlock()
		case <-lt.done:
			return
		}
	}
}
