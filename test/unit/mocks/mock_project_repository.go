package mocks

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// MockProjectRepository implements repositories.ProjectRepository for testing.
// Each method is a function field that you can configure per test.
type MockProjectRepository struct {
	CreateFn         func(ctx context.Context, project *entities.Project) error
	GetByIDFn        func(ctx context.Context, id uuid.UUID) (*entities.Project, error)
	ListByClientIDFn func(ctx context.Context, clientID uuid.UUID, pg pagination.Pagination) ([]*entities.Project, int64, error)
	UpdateFn         func(ctx context.Context, project *entities.Project) error
	DeleteFn         func(ctx context.Context, id uuid.UUID) error
}

func (m *MockProjectRepository) Create(ctx context.Context, project *entities.Project) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, project)
	}
	return nil
}

func (m *MockProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Project, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockProjectRepository) ListByClientID(ctx context.Context, clientID uuid.UUID, pg pagination.Pagination) ([]*entities.Project, int64, error) {
	if m.ListByClientIDFn != nil {
		return m.ListByClientIDFn(ctx, clientID, pg)
	}
	return nil, 0, nil
}

func (m *MockProjectRepository) Update(ctx context.Context, project *entities.Project) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, project)
	}
	return nil
}

func (m *MockProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
