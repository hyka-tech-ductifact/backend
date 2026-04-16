package repositories

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// PieceRepository is the outbound port for piece persistence.
// It is defined in the domain but implemented in infrastructure.
type PieceRepository interface {
	Create(ctx context.Context, piece *entities.Piece) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Piece, error)
	ListByOrderID(ctx context.Context, orderID uuid.UUID, pg pagination.Pagination) ([]*entities.Piece, int64, error)
	Update(ctx context.Context, piece *entities.Piece) error
	Delete(ctx context.Context, id uuid.UUID) error
}
