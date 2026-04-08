package auth_test

import (
	"testing"
	"time"

	"ductifact/internal/infrastructure/auth"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// MemoryBlacklist
// =============================================================================

func TestMemoryBlacklist_Add_MakesTokenBlacklisted(t *testing.T) {
	bl := auth.NewMemoryBlacklist(1 * time.Minute)
	defer bl.Stop()

	bl.Add("token-123", 1*time.Hour)

	assert.True(t, bl.IsBlacklisted("token-123"))
}

func TestMemoryBlacklist_IsBlacklisted_ReturnsFalseForUnknownToken(t *testing.T) {
	bl := auth.NewMemoryBlacklist(1 * time.Minute)
	defer bl.Stop()

	assert.False(t, bl.IsBlacklisted("never-added"))
}

func TestMemoryBlacklist_ExpiredToken_IsNotBlacklisted(t *testing.T) {
	bl := auth.NewMemoryBlacklist(1 * time.Minute)
	defer bl.Stop()

	// Add with a tiny expiry
	bl.Add("short-lived", 1*time.Millisecond)

	// Wait for it to expire
	time.Sleep(5 * time.Millisecond)

	assert.False(t, bl.IsBlacklisted("short-lived"))
}

func TestMemoryBlacklist_MultipleTokens(t *testing.T) {
	bl := auth.NewMemoryBlacklist(1 * time.Minute)
	defer bl.Stop()

	bl.Add("token-a", 1*time.Hour)
	bl.Add("token-b", 1*time.Hour)

	assert.True(t, bl.IsBlacklisted("token-a"))
	assert.True(t, bl.IsBlacklisted("token-b"))
	assert.False(t, bl.IsBlacklisted("token-c"))
}
