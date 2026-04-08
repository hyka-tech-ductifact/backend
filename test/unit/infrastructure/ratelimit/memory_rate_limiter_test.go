package ratelimit_test

import (
	"testing"
	"time"

	"ductifact/internal/infrastructure/ratelimit"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Allow
// =============================================================================

func TestAllow_FirstRequest_ReturnsTrue(t *testing.T) {
	rl := ratelimit.NewMemoryRateLimiter(5, 1*time.Minute, 10*time.Minute)
	defer rl.Stop()

	assert.True(t, rl.Allow("ip:192.168.1.1"))
}

func TestAllow_WithinLimit_AllReturnTrue(t *testing.T) {
	rl := ratelimit.NewMemoryRateLimiter(3, 1*time.Minute, 10*time.Minute)
	defer rl.Stop()

	assert.True(t, rl.Allow("ip:10.0.0.1"))
	assert.True(t, rl.Allow("ip:10.0.0.1"))
	assert.True(t, rl.Allow("ip:10.0.0.1"))
}

func TestAllow_ExceedsLimit_ReturnsFalse(t *testing.T) {
	rl := ratelimit.NewMemoryRateLimiter(3, 1*time.Minute, 10*time.Minute)
	defer rl.Stop()

	rl.Allow("ip:10.0.0.1")
	rl.Allow("ip:10.0.0.1")
	rl.Allow("ip:10.0.0.1")

	assert.False(t, rl.Allow("ip:10.0.0.1"))
}

func TestAllow_DifferentKeys_IndependentCounters(t *testing.T) {
	rl := ratelimit.NewMemoryRateLimiter(2, 1*time.Minute, 10*time.Minute)
	defer rl.Stop()

	// Key A uses up its quota
	rl.Allow("ip:10.0.0.1")
	rl.Allow("ip:10.0.0.1")
	assert.False(t, rl.Allow("ip:10.0.0.1"))

	// Key B is unaffected
	assert.True(t, rl.Allow("ip:10.0.0.2"))
	assert.True(t, rl.Allow("ip:10.0.0.2"))
}

func TestAllow_WindowExpires_CounterResets(t *testing.T) {
	rl := ratelimit.NewMemoryRateLimiterWithClock(2, 50*time.Millisecond, 10*time.Minute)
	defer rl.Stop()

	rl.Allow("ip:10.0.0.1")
	rl.Allow("ip:10.0.0.1")
	assert.False(t, rl.Allow("ip:10.0.0.1"))

	// Wait for the window to expire
	time.Sleep(60 * time.Millisecond)

	// Counter should be reset
	assert.True(t, rl.Allow("ip:10.0.0.1"))
}

func TestAllow_LimitOfOne_RejectSecondRequest(t *testing.T) {
	rl := ratelimit.NewMemoryRateLimiter(1, 1*time.Minute, 10*time.Minute)
	defer rl.Stop()

	assert.True(t, rl.Allow("user:abc"))
	assert.False(t, rl.Allow("user:abc"))
}

// =============================================================================
// Stop
// =============================================================================

func TestStop_DoesNotPanic(t *testing.T) {
	rl := ratelimit.NewMemoryRateLimiter(10, 1*time.Minute, 100*time.Millisecond)

	assert.NotPanics(t, func() {
		rl.Stop()
	})
}

// =============================================================================
// Concurrency
// =============================================================================

func TestAllow_ConcurrentAccess_DoesNotPanic(t *testing.T) {
	rl := ratelimit.NewMemoryRateLimiter(1000, 1*time.Minute, 10*time.Minute)
	defer rl.Stop()

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				rl.Allow("ip:concurrent")
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}
