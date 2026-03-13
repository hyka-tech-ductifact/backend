package ports

import "github.com/google/uuid"

// TokenProvider is the outbound port for JWT operations.
// It is defined as an interface so the auth service doesn't depend on
// a specific JWT library — the implementation lives in infrastructure.
type TokenProvider interface {
	GenerateToken(userID uuid.UUID, email string) (string, error)
	ValidateToken(tokenString string) (*TokenClaims, error)
}

// TokenClaims holds the data extracted from a valid token.
type TokenClaims struct {
	UserID uuid.UUID
	Email  string
}
