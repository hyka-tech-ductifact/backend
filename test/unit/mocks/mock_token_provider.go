package mocks

import (
	"ductifact/internal/application/ports"

	"github.com/google/uuid"
)

// MockTokenProvider implements ports.TokenProvider for testing.
// Each method is a function field that you can configure per test.
type MockTokenProvider struct {
	GenerateTokenFn func(userID uuid.UUID, email string) (string, error)
	ValidateTokenFn func(tokenString string) (*ports.TokenClaims, error)
}

func (m *MockTokenProvider) GenerateToken(userID uuid.UUID, email string) (string, error) {
	if m.GenerateTokenFn != nil {
		return m.GenerateTokenFn(userID, email)
	}
	return "mock-token", nil
}

func (m *MockTokenProvider) ValidateToken(tokenString string) (*ports.TokenClaims, error) {
	if m.ValidateTokenFn != nil {
		return m.ValidateTokenFn(tokenString)
	}
	return nil, nil
}
