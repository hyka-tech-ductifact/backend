package usecases

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// PieceDefinitionService is the inbound port for piece definition operations.
// Inbound adapters (HTTP handlers, CLI, etc.) depend on this interface.
type PieceDefinitionService interface {
	CreatePieceDefinition(ctx context.Context, userID uuid.UUID, params entities.CreatePieceDefParams) (*entities.PieceDefinition, error)
	GetPieceDefinitionByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.PieceDefinition, error)
	ListPieceDefinitions(ctx context.Context, userID uuid.UUID, pg pagination.Pagination) (pagination.Result[*entities.PieceDefinition], error)
	UpdatePieceDefinition(ctx context.Context, id uuid.UUID, userID uuid.UUID, params entities.UpdatePieceDefParams) (*entities.PieceDefinition, error)
	DeletePieceDefinition(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
