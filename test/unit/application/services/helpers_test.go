package services_test

import (
	"context"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"
	"ductifact/test/unit/mocks"

	"github.com/google/uuid"
)

// --- Entity factories ---

// newTestUser creates a User with sensible defaults and timestamps in the past.
func newTestUser() *entities.User {
	return &entities.User{
		ID:        uuid.New(),
		Name:      "Juan",
		Email:     "juan@example.com",
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}
}

// newTestClient creates a Client with sensible defaults and timestamps in the past.
func newTestClient(userID uuid.UUID) *entities.Client {
	return &entities.Client{
		ID:          uuid.New(),
		Name:        "Acme Corp",
		Phone:       "+34 612 345 678",
		Email:       "contact@acme.com",
		Description: "Main partner",
		UserID:      userID,
		CreatedAt:   time.Now().Add(-time.Hour),
		UpdatedAt:   time.Now().Add(-time.Hour),
	}
}

// --- Mock helpers ---

// userRepoReturning builds a MockUserRepository whose GetByIDFn always returns
// a copy of the given user (or ErrNotFound if user is nil).
func userRepoReturning(user *entities.User) *mocks.MockUserRepository {
	return &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			if user == nil {
				return nil, repositories.ErrNotFound
			}
			cp := *user
			return &cp, nil
		},
		UpdateFn: func(ctx context.Context, u *entities.User) error {
			return nil
		},
	}
}

// clientRepoReturning builds a MockClientRepository whose GetByIDFn always returns
// a copy of the given client (or ErrNotFound if client is nil).
func clientRepoReturning(client *entities.Client) *mocks.MockClientRepository {
	return &mocks.MockClientRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
			if client == nil {
				return nil, repositories.ErrNotFound
			}
			cp := *client
			return &cp, nil
		},
		UpdateFn: func(ctx context.Context, c *entities.Client) error {
			return nil
		},
	}
}

// strPtr returns a pointer to the given string. Useful for optional update fields.
func strPtr(s string) *string {
	return &s
}
