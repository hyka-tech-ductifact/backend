package entities

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// TokenType identifies the purpose of a one-time token.
type TokenType string

const (
	TokenTypeEmailVerification TokenType = "email_verification"
	TokenTypePasswordReset     TokenType = "password_reset"
)

// OneTimeToken represents a single-use, expiring token tied to a user.
// It is generic by design — the Type field determines its purpose
// (email verification, password reset, etc.).
type OneTimeToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	Type      TokenType
	ExpiresAt time.Time
	CreatedAt time.Time
}

// NewOneTimeToken creates a token of the given type for the specified user.
// The token is a 32-byte cryptographically random hex string (64 chars).
// It expires after the given duration.
func NewOneTimeToken(userID uuid.UUID, tokenType TokenType, ttl time.Duration) (*OneTimeToken, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}

	now := time.Now()
	return &OneTimeToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     hex.EncodeToString(tokenBytes),
		Type:      tokenType,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}, nil
}

// IsExpired returns true if the token has passed its expiration time.
func (t *OneTimeToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}
