package usecases

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// PieceService is the inbound port for piece operations.
// Inbound adapters (HTTP handlers, CLI, etc.) depend on this interface.
type PieceService interface {
	CreatePiece(ctx context.Context, userID uuid.UUID, params entities.CreatePieceParams) (*entities.Piece, error)
	GetPieceByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Piece, error)
	ListPiecesByOrderID(ctx context.Context, orderID uuid.UUID, userID uuid.UUID, pg pagination.Pagination) (pagination.Result[*entities.Piece], error)
	UpdatePiece(ctx context.Context, id uuid.UUID, userID uuid.UUID, params entities.UpdatePieceParams) (*entities.Piece, error)
	DeletePiece(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
