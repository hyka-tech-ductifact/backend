package mocks

import (
	"context"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// MockUserRepository implements repositories.UserRepository for testing.
// Each method is a function field that you can configure per test.
type MockUserRepository struct {
	CreateFn     func(ctx context.Context, user *entities.User) error
	GetByIDFn    func(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetByEmailFn func(ctx context.Context, email string) (*entities.User, error)
	UpdateFn     func(ctx context.Context, user *entities.User) error
	DeleteFn     func(ctx context.Context, id uuid.UUID) error
}

func (m *MockUserRepository) Create(ctx context.Context, user *entities.User) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *entities.User) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
