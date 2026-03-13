package repositories

import (
	"context"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// ClientRepository is the outbound port for client persistence.
// It is defined in the domain but implemented in infrastructure.
type ClientRepository interface {
	Create(ctx context.Context, client *entities.Client) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Client, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Client, error)
	Update(ctx context.Context, client *entities.Client) error
	Delete(ctx context.Context, id uuid.UUID) error
}
