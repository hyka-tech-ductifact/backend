package usecases

import (
	"context"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// UserService is the inbound port for user operations.
// Inbound adapters (HTTP handlers, CLI, etc.) depend on this interface.
// Note: User creation is handled by AuthService.Register (authentication flow).
type UserService interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, name, email, locale *string) (*entities.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID, cascade bool) error
}
