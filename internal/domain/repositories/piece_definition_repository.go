package repositories

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// PieceDefinitionRepository is the outbound port for piece definition persistence.
// It is defined in the domain but implemented in infrastructure.
type PieceDefinitionRepository interface {
	Create(ctx context.Context, def *entities.PieceDefinition) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.PieceDefinition, error)
	GetByIDForOwner(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.PieceDefinition, error)
	ListByUserID(
		ctx context.Context,
		userID uuid.UUID,
		includeArchived bool,
		pg pagination.Pagination,
	) ([]*entities.PieceDefinition, int64, error)
	Update(ctx context.Context, def *entities.PieceDefinition) error
	Delete(ctx context.Context, id uuid.UUID) error
	Archive(ctx context.Context, id uuid.UUID) error
	Unarchive(ctx context.Context, id uuid.UUID) error
}
