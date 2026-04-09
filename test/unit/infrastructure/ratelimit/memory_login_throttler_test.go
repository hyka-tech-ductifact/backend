package ratelimit_test

import (
	"testing"
	"time"

	"ductifact/internal/infrastructure/ratelimit"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// IsBlocked
// =============================================================================

func TestThrottler_NewKey_IsNotBlocked(t *testing.T) {
	lt := ratelimit.NewMemoryLoginThrottler(5, 15*time.Minute, 15*time.Minute, 10*time.Minute)
	defer lt.Stop()

	assert.False(t, lt.IsBlocked("juan@example.com"))
}

func TestThrottler_BelowThreshold_IsNotBlocked(t *testing.T) {
	lt := ratelimit.NewMemoryLoginThrottler(3, 15*time.Minute, 15*time.Minute, 10*time.Minute)
	defer lt.Stop()

	lt.RecordFailure("juan@example.com")
	lt.RecordFailure("juan@example.com")

	assert.False(t, lt.IsBlocked("juan@example.com"))
}

func TestThrottler_AtThreshold_IsBlocked(t *testing.T) {
	lt := ratelimit.NewMemoryLoginThrottler(3, 15*time.Minute, 15*time.Minute, 10*time.Minute)
	defer lt.Stop()

	lt.RecordFailure("juan@example.com")
	lt.RecordFailure("juan@example.com")
	lt.RecordFailure("juan@example.com")

	assert.True(t, lt.IsBlocked("juan@example.com"))
}

func TestThrottler_AboveThreshold_StillBlocked(t *testing.T) {
	lt := ratelimit.NewMemoryLoginThrottler(3, 15*time.Minute, 15*time.Minute, 10*time.Minute)
	defer lt.Stop()

	for i := 0; i < 10; i++ {
		lt.RecordFailure("juan@example.com")
	}

	assert.True(t, lt.IsBlocked("juan@example.com"))
}

// =============================================================================
// RecordFailure — different keys are independent
// =============================================================================

func TestThrottler_DifferentKeys_AreIndependent(t *testing.T) {
	lt := ratelimit.NewMemoryLoginThrottler(2, 15*time.Minute, 15*time.Minute, 10*time.Minute)
	defer lt.Stop()

	lt.RecordFailure("a@example.com")
	lt.RecordFailure("a@example.com")

	assert.True(t, lt.IsBlocked("a@example.com"))
	assert.False(t, lt.IsBlocked("b@example.com"))
}

// =============================================================================
// Reset
// =============================================================================

func TestThrottler_Reset_ClearsFailures(t *testing.T) {
	lt := ratelimit.NewMemoryLoginThrottler(3, 15*time.Minute, 15*time.Minute, 10*time.Minute)
	defer lt.Stop()

	lt.RecordFailure("juan@example.com")
	lt.RecordFailure("juan@example.com")
	lt.RecordFailure("juan@example.com")
	assert.True(t, lt.IsBlocked("juan@example.com"))

	lt.Reset("juan@example.com")

	assert.False(t, lt.IsBlocked("juan@example.com"))
}

func TestThrottler_Reset_NonexistentKey_DoesNotPanic(t *testing.T) {
	lt := ratelimit.NewMemoryLoginThrottler(3, 15*time.Minute, 15*time.Minute, 10*time.Minute)
	defer lt.Stop()

	assert.NotPanics(t, func() {
		lt.Reset("nonexistent@example.com")
	})
}

// =============================================================================
// Lockout expiry
// =============================================================================

func TestThrottler_LockoutExpires_IsUnblocked(t *testing.T) {
	lt := ratelimit.NewMemoryLoginThrottler(2, 15*time.Minute, 50*time.Millisecond, 10*time.Minute)
	defer lt.Stop()

	lt.RecordFailure("juan@example.com")
	lt.RecordFailure("juan@example.com")
	assert.True(t, lt.IsBlocked("juan@example.com"))

	// Wait for lockout to expire
	time.Sleep(60 * time.Millisecond)

	assert.False(t, lt.IsBlocked("juan@example.com"))
}

func TestThrottler_AfterLockoutExpires_CanAccumulateAgain(t *testing.T) {
	lt := ratelimit.NewMemoryLoginThrottler(2, 15*time.Minute, 50*time.Millisecond, 10*time.Minute)
	defer lt.Stop()

	// First lockout
	lt.RecordFailure("juan@example.com")
	lt.RecordFailure("juan@example.com")
	assert.True(t, lt.IsBlocked("juan@example.com"))

	// Wait for lockout to expire
	time.Sleep(60 * time.Millisecond)

	// Should be able to fail again and get locked again
	lt.RecordFailure("juan@example.com")
	assert.False(t, lt.IsBlocked("juan@example.com"))

	lt.RecordFailure("juan@example.com")
	assert.True(t, lt.IsBlocked("juan@example.com"))
}

// =============================================================================
// Stop
// =============================================================================

func TestThrottler_Stop_DoesNotPanic(t *testing.T) {
	lt := ratelimit.NewMemoryLoginThrottler(5, 15*time.Minute, 15*time.Minute, 100*time.Millisecond)

	assert.NotPanics(t, func() {
		lt.Stop()
	})
}

// =============================================================================
// Concurrency
// =============================================================================

func TestThrottler_ConcurrentAccess_DoesNotPanic(t *testing.T) {
	lt := ratelimit.NewMemoryLoginThrottler(1000, 15*time.Minute, 15*time.Minute, 10*time.Minute)
	defer lt.Stop()

	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func() {
			for j := 0; j < 20; j++ {
				lt.RecordFailure("concurrent@example.com")
				lt.IsBlocked("concurrent@example.com")
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
	}
}
