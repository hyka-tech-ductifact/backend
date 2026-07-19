package ports

import (
	"time"

	"github.com/google/uuid"
)

// BearerTokenType is the OAuth-compatible token type returned to API clients.
const BearerTokenType = "Bearer"

// TokenProvider is the outbound port for JWT operations.
// It is defined as an interface so the auth service doesn't depend on
// a specific JWT library — the implementation lives in infrastructure.
type TokenProvider interface {
	GenerateTokenPair(userID uuid.UUID, email string) (*TokenPair, error)
	ValidateToken(tokenString string) (*TokenClaims, error)
	ValidateRefreshToken(tokenString string) (*TokenClaims, error)
}

// TokenPair holds the access and refresh tokens returned after authentication.
type TokenPair struct {
	AccessToken     string
	RefreshToken    string
	TokenType       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// TokenClaims holds the data extracted from a valid token.
type TokenClaims struct {
	UserID uuid.UUID
	Email  string
}
