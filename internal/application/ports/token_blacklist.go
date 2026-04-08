package ports

import "time"

// TokenBlacklist is the outbound port for revoking tokens.
// Tokens are stored until their natural expiry, then cleaned up.
// This enables logout functionality in a stateless JWT system.
type TokenBlacklist interface {
	// Add marks a token as revoked. It will be automatically removed
	// after the given duration (matching the token's remaining lifetime).
	Add(token string, expiry time.Duration)

	// IsBlacklisted returns true if the token has been revoked.
	IsBlacklisted(token string) bool
}
