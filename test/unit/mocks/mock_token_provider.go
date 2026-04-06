package mocks

import (
	"ductifact/internal/application/ports"

	"github.com/google/uuid"
)

// MockTokenProvider implements ports.TokenProvider for testing.
// Each method is a function field that you can configure per test.
type MockTokenProvider struct {
	GenerateTokenPairFn    func(userID uuid.UUID, email string) (*ports.TokenPair, error)
	ValidateTokenFn        func(tokenString string) (*ports.TokenClaims, error)
	ValidateRefreshTokenFn func(tokenString string) (*ports.TokenClaims, error)
}

func (m *MockTokenProvider) GenerateTokenPair(userID uuid.UUID, email string) (*ports.TokenPair, error) {
	if m.GenerateTokenPairFn != nil {
		return m.GenerateTokenPairFn(userID, email)
	}
	return &ports.TokenPair{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
	}, nil
}

func (m *MockTokenProvider) ValidateToken(tokenString string) (*ports.TokenClaims, error) {
	if m.ValidateTokenFn != nil {
		return m.ValidateTokenFn(tokenString)
	}
	return nil, nil
}

func (m *MockTokenProvider) ValidateRefreshToken(tokenString string) (*ports.TokenClaims, error) {
	if m.ValidateRefreshTokenFn != nil {
		return m.ValidateRefreshTokenFn(tokenString)
	}
	return nil, nil
}
