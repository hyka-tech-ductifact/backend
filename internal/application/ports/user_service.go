package ports

import (
	"context"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// UserService is the inbound port for user operations.
// Inbound adapters (HTTP handlers, CLI, etc.) depend on this interface.
type UserService interface {
	CreateUser(ctx context.Context, name, email string) (*entities.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, name, email *string) (*entities.User, error)
}
