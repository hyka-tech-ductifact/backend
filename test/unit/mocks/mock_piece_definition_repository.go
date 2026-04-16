package mocks

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// MockPieceDefinitionRepository implements repositories.PieceDefinitionRepository for testing.
// Each method is a function field that you can configure per test.
type MockPieceDefinitionRepository struct {
	CreateFn       func(ctx context.Context, def *entities.PieceDefinition) error
	GetByIDFn      func(ctx context.Context, id uuid.UUID) (*entities.PieceDefinition, error)
	ListByUserIDFn func(ctx context.Context, userID uuid.UUID, pg pagination.Pagination) ([]*entities.PieceDefinition, int64, error)
	UpdateFn       func(ctx context.Context, def *entities.PieceDefinition) error
	DeleteFn       func(ctx context.Context, id uuid.UUID) error
}

func (m *MockPieceDefinitionRepository) Create(ctx context.Context, def *entities.PieceDefinition) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, def)
	}
	return nil
}

func (m *MockPieceDefinitionRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.PieceDefinition, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockPieceDefinitionRepository) ListByUserID(ctx context.Context, userID uuid.UUID, pg pagination.Pagination) ([]*entities.PieceDefinition, int64, error) {
	if m.ListByUserIDFn != nil {
		return m.ListByUserIDFn(ctx, userID, pg)
	}
	return nil, 0, nil
}

func (m *MockPieceDefinitionRepository) Update(ctx context.Context, def *entities.PieceDefinition) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, def)
	}
	return nil
}

func (m *MockPieceDefinitionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
