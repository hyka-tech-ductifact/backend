package repositories

import (
	"context"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// UserRepository is the outbound port for user persistence.
// It is defined in the domain but implemented in infrastructure.
type UserRepository interface {
	Create(ctx context.Context, user *entities.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	Update(ctx context.Context, user *entities.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}
