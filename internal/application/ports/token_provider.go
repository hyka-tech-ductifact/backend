package ports

import "github.com/google/uuid"

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
	AccessToken  string
	RefreshToken string
}

// TokenClaims holds the data extracted from a valid token.
type TokenClaims struct {
	UserID uuid.UUID
	Email  string
}
