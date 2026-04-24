package repositories

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// OrderRepository is the outbound port for order persistence.
// It is defined in the domain but implemented in infrastructure.
type OrderRepository interface {
	Create(ctx context.Context, order *entities.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Order, error)
	GetByIDForOwner(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Order, error)
	ListByProjectID(ctx context.Context, projectID uuid.UUID, pg pagination.Pagination) ([]*entities.Order, int64, error)
	CountByProjectID(ctx context.Context, projectID uuid.UUID) (int64, error)
	Update(ctx context.Context, order *entities.Order) error
	Delete(ctx context.Context, id uuid.UUID) error
}
