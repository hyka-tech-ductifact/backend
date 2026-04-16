package mocks

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// MockPieceRepository implements repositories.PieceRepository for testing.
// Each method is a function field that you can configure per test.
type MockPieceRepository struct {
	CreateFn          func(ctx context.Context, piece *entities.Piece) error
	GetByIDFn         func(ctx context.Context, id uuid.UUID) (*entities.Piece, error)
	GetByIDForOwnerFn func(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Piece, error)
	ListByOrderIDFn   func(ctx context.Context, orderID uuid.UUID, pg pagination.Pagination) ([]*entities.Piece, int64, error)
	UpdateFn          func(ctx context.Context, piece *entities.Piece) error
	DeleteFn          func(ctx context.Context, id uuid.UUID) error
}

func (m *MockPieceRepository) Create(ctx context.Context, piece *entities.Piece) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, piece)
	}
	return nil
}

func (m *MockPieceRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Piece, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockPieceRepository) GetByIDForOwner(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Piece, error) {
	if m.GetByIDForOwnerFn != nil {
		return m.GetByIDForOwnerFn(ctx, id, ownerID)
	}
	return nil, nil
}

func (m *MockPieceRepository) ListByOrderID(ctx context.Context, orderID uuid.UUID, pg pagination.Pagination) ([]*entities.Piece, int64, error) {
	if m.ListByOrderIDFn != nil {
		return m.ListByOrderIDFn(ctx, orderID, pg)
	}
	return nil, 0, nil
}

func (m *MockPieceRepository) Update(ctx context.Context, piece *entities.Piece) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, piece)
	}
	return nil
}

func (m *MockPieceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
