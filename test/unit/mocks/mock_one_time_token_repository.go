package mocks

import (
	"context"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// MockOneTimeTokenRepository implements repositories.OneTimeTokenRepository for testing.
type MockOneTimeTokenRepository struct {
	CreateFn                func(ctx context.Context, token *entities.OneTimeToken) error
	GetByTokenFn            func(ctx context.Context, token string, tokenType entities.TokenType) (*entities.OneTimeToken, error)
	DeleteByUserIDAndTypeFn func(ctx context.Context, userID uuid.UUID, tokenType entities.TokenType) error
	Created                 []*entities.OneTimeToken // captures all tokens created (when CreateFn is nil)
}

func (m *MockOneTimeTokenRepository) Create(ctx context.Context, token *entities.OneTimeToken) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, token)
	}
	m.Created = append(m.Created, token)
	return nil
}

func (m *MockOneTimeTokenRepository) GetByToken(ctx context.Context, token string, tokenType entities.TokenType) (*entities.OneTimeToken, error) {
	if m.GetByTokenFn != nil {
		return m.GetByTokenFn(ctx, token, tokenType)
	}
	return nil, nil
}

func (m *MockOneTimeTokenRepository) DeleteByUserIDAndType(ctx context.Context, userID uuid.UUID, tokenType entities.TokenType) error {
	if m.DeleteByUserIDAndTypeFn != nil {
		return m.DeleteByUserIDAndTypeFn(ctx, userID, tokenType)
	}
	return nil
}
