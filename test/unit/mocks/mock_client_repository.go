package mocks

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// MockClientRepository implements repositories.ClientRepository for testing.
// Each method is a function field that you can configure per test.
type MockClientRepository struct {
	CreateFn          func(ctx context.Context, client *entities.Client) error
	GetByIDFn         func(ctx context.Context, id uuid.UUID) (*entities.Client, error)
	GetByIDForOwnerFn func(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Client, error)
	ListByUserIDFn    func(ctx context.Context, userID uuid.UUID, pg pagination.Pagination) ([]*entities.Client, int64, error)
	CountByUserIDFn   func(ctx context.Context, userID uuid.UUID) (int64, error)
	UpdateFn          func(ctx context.Context, client *entities.Client) error
	DeleteFn          func(ctx context.Context, id uuid.UUID) error
}

func (m *MockClientRepository) Create(ctx context.Context, client *entities.Client) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, client)
	}
	return nil
}

func (m *MockClientRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockClientRepository) GetByIDForOwner(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Client, error) {
	if m.GetByIDForOwnerFn != nil {
		return m.GetByIDForOwnerFn(ctx, id, ownerID)
	}
	return nil, nil
}

func (m *MockClientRepository) ListByUserID(ctx context.Context, userID uuid.UUID, pg pagination.Pagination) ([]*entities.Client, int64, error) {
	if m.ListByUserIDFn != nil {
		return m.ListByUserIDFn(ctx, userID, pg)
	}
	return nil, 0, nil
}

func (m *MockClientRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	if m.CountByUserIDFn != nil {
		return m.CountByUserIDFn(ctx, userID)
	}
	return 0, nil
}

func (m *MockClientRepository) Update(ctx context.Context, client *entities.Client) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, client)
	}
	return nil
}

func (m *MockClientRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
