package entities_test

import (
	"testing"
	"time"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOneTimeToken_ReturnsValidToken(t *testing.T) {
	userID := uuid.New()

	token, err := entities.NewOneTimeToken(userID, entities.TokenTypeEmailVerification, 24*time.Hour)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, token.ID)
	assert.Equal(t, userID, token.UserID)
	assert.Equal(t, entities.TokenTypeEmailVerification, token.Type)
	assert.Len(t, token.Token, 64, "token should be 64 hex chars (32 bytes)")
	assert.False(t, token.CreatedAt.IsZero())
	assert.False(t, token.ExpiresAt.IsZero())
}

func TestNewOneTimeToken_TokenIsHexEncoded(t *testing.T) {
	token, err := entities.NewOneTimeToken(uuid.New(), entities.TokenTypePasswordReset, time.Hour)

	require.NoError(t, err)
	for _, c := range token.Token {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"token should only contain hex characters, got %c", c)
	}
}

func TestNewOneTimeToken_ExpiresAtMatchesTTL(t *testing.T) {
	before := time.Now()
	token, err := entities.NewOneTimeToken(uuid.New(), entities.TokenTypeEmailVerification, 2*time.Hour)
	after := time.Now()

	require.NoError(t, err)
	assert.True(t, token.ExpiresAt.After(before.Add(2*time.Hour-time.Second)))
	assert.True(t, token.ExpiresAt.Before(after.Add(2*time.Hour+time.Second)))
}

func TestNewOneTimeToken_GeneratesUniqueTokens(t *testing.T) {
	userID := uuid.New()
	token1, _ := entities.NewOneTimeToken(userID, entities.TokenTypeEmailVerification, time.Hour)
	token2, _ := entities.NewOneTimeToken(userID, entities.TokenTypeEmailVerification, time.Hour)

	assert.NotEqual(t, token1.Token, token2.Token)
	assert.NotEqual(t, token1.ID, token2.ID)
}

func TestIsExpired_WhenNotExpired_ReturnsFalse(t *testing.T) {
	token, err := entities.NewOneTimeToken(uuid.New(), entities.TokenTypeEmailVerification, time.Hour)

	require.NoError(t, err)
	assert.False(t, token.IsExpired())
}

func TestIsExpired_WhenExpired_ReturnsTrue(t *testing.T) {
	token := &entities.OneTimeToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Token:     "abc123",
		Type:      entities.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(-1 * time.Minute),
		CreatedAt: time.Now().Add(-25 * time.Hour),
	}

	assert.True(t, token.IsExpired())
}

func TestTokenType_Constants(t *testing.T) {
	assert.Equal(t, entities.TokenType("email_verification"), entities.TokenTypeEmailVerification)
	assert.Equal(t, entities.TokenType("password_reset"), entities.TokenTypePasswordReset)
}
