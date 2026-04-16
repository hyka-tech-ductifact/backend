package services

import (
	"context"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
)

// pieceService implements usecases.PieceService.
// Unexported struct: can only be created via NewPieceService.
type pieceService struct {
	pieceRepo    repositories.PieceRepository
	pieceDefRepo repositories.PieceDefinitionRepository
	orderRepo    repositories.OrderRepository
}

// NewPieceService creates a new PieceService.
// The order repository is needed to verify ownership during Create and List.
// The piece definition repository is needed to validate dimensions.
func NewPieceService(
	pieceRepo repositories.PieceRepository,
	pieceDefRepo repositories.PieceDefinitionRepository,
	orderRepo repositories.OrderRepository,
) *pieceService {
	return &pieceService{
		pieceRepo:    pieceRepo,
		pieceDefRepo: pieceDefRepo,
		orderRepo:    orderRepo,
	}
}

// CreatePiece orchestrates piece creation:
// 1. Verify the owning order belongs to the user.
// 2. Verify the piece definition is accessible to the user.
// 3. Build the domain entity (which validates all fields).
// 4. Validate dimensions against the definition schema.
// 5. Persist via repository.
func (s *pieceService) CreatePiece(ctx context.Context, userID uuid.UUID, params entities.CreatePieceParams) (*entities.Piece, error) {
	// Step 1: Verify order ownership
	_, err := s.orderRepo.GetByIDForOwner(ctx, params.OrderID, userID)
	if err != nil {
		return nil, err
	}

	// Step 2: Verify piece definition accessibility
	def, err := s.pieceDefRepo.GetByIDForOwner(ctx, params.DefinitionID, userID)
	if err != nil {
		return nil, err
	}

	// Step 3: Domain entity validates its own invariants
	piece, err := entities.NewPiece(params)
	if err != nil {
		return nil, err
	}

	// Step 4: Validate dimensions against definition schema
	if err := piece.ValidateAgainst(def); err != nil {
		return nil, err
	}

	// Step 5: Persist
	if err := s.pieceRepo.Create(ctx, piece); err != nil {
		return nil, err
	}

	return piece, nil
}

// GetPieceByID retrieves a piece by ID, ensuring it belongs to the given user.
func (s *pieceService) GetPieceByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Piece, error) {
	return s.pieceRepo.GetByIDForOwner(ctx, id, userID)
}

// ListPiecesByOrderID retrieves a paginated list of pieces for an order,
// ensuring the order belongs to the given user.
func (s *pieceService) ListPiecesByOrderID(ctx context.Context, orderID uuid.UUID, userID uuid.UUID, pg pagination.Pagination) (pagination.Result[*entities.Piece], error) {
	_, err := s.orderRepo.GetByIDForOwner(ctx, orderID, userID)
	if err != nil {
		return pagination.Result[*entities.Piece]{}, err
	}

	pieces, totalItems, err := s.pieceRepo.ListByOrderID(ctx, orderID, pg)
	if err != nil {
		return pagination.Result[*entities.Piece]{}, err
	}

	return pagination.NewResult(pieces, pg, totalItems), nil
}

// UpdatePiece applies a partial update to an existing piece.
// Only non-nil fields in params are updated. Ensures the piece belongs to the given user.
// If dimensions change, they are re-validated against the piece definition schema.
func (s *pieceService) UpdatePiece(ctx context.Context, id uuid.UUID, userID uuid.UUID, params entities.UpdatePieceParams) (*entities.Piece, error) {
	piece, err := s.pieceRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Nothing to update
	if !params.HasChanges() {
		return piece, nil
	}

	if params.Title != nil {
		if err := piece.SetTitle(*params.Title); err != nil {
			return nil, err
		}
	}
	if params.Quantity != nil {
		if err := piece.SetQuantity(*params.Quantity); err != nil {
			return nil, err
		}
	}
	if params.Dimensions != nil {
		piece.Dimensions = *params.Dimensions

		def, err := s.pieceDefRepo.GetByID(ctx, piece.DefinitionID)
		if err != nil {
			return nil, err
		}
		if err := piece.ValidateAgainst(def); err != nil {
			return nil, err
		}
	}

	// Update timestamp and persist
	piece.UpdatedAt = time.Now()
	if err := s.pieceRepo.Update(ctx, piece); err != nil {
		return nil, err
	}

	return piece, nil
}

// DeletePiece removes a piece, ensuring it belongs to the given user.
func (s *pieceService) DeletePiece(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	piece, err := s.pieceRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return err
	}

	return s.pieceRepo.Delete(ctx, piece.ID)
}
