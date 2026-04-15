package mocks

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// MockOrderRepository implements repositories.OrderRepository for testing.
// Each method is a function field that you can configure per test.
type MockOrderRepository struct {
	CreateFn          func(ctx context.Context, order *entities.Order) error
	GetByIDFn         func(ctx context.Context, id uuid.UUID) (*entities.Order, error)
	ListByProjectIDFn func(ctx context.Context, projectID uuid.UUID, pg pagination.Pagination) ([]*entities.Order, int64, error)
	UpdateFn          func(ctx context.Context, order *entities.Order) error
	DeleteFn          func(ctx context.Context, id uuid.UUID) error
}

func (m *MockOrderRepository) Create(ctx context.Context, order *entities.Order) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, order)
	}
	return nil
}

func (m *MockOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Order, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockOrderRepository) ListByProjectID(ctx context.Context, projectID uuid.UUID, pg pagination.Pagination) ([]*entities.Order, int64, error) {
	if m.ListByProjectIDFn != nil {
		return m.ListByProjectIDFn(ctx, projectID, pg)
	}
	return nil, 0, nil
}

func (m *MockOrderRepository) Update(ctx context.Context, order *entities.Order) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, order)
	}
	return nil
}

func (m *MockOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
