package services

import (
	"context"
	"errors"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
)

// --- Application-level errors ---

var (
	ErrPieceNotFound = errors.New("piece not found")
	ErrPieceNotOwned = errors.New("piece does not belong to this order")
)

// pieceService implements usecases.PieceService.
// Unexported struct: can only be created via NewPieceService.
type pieceService struct {
	pieceRepo    repositories.PieceRepository
	pieceDefRepo repositories.PieceDefinitionRepository
	orderRepo    repositories.OrderRepository
	projectRepo  repositories.ProjectRepository
	clientRepo   repositories.ClientRepository
}

// NewPieceService creates a new PieceService.
// It receives all repositories needed to verify the full ownership chain:
// User → Client → Project → Order → Piece.
func NewPieceService(
	pieceRepo repositories.PieceRepository,
	pieceDefRepo repositories.PieceDefinitionRepository,
	orderRepo repositories.OrderRepository,
	projectRepo repositories.ProjectRepository,
	clientRepo repositories.ClientRepository,
) *pieceService {
	return &pieceService{
		pieceRepo:    pieceRepo,
		pieceDefRepo: pieceDefRepo,
		orderRepo:    orderRepo,
		projectRepo:  projectRepo,
		clientRepo:   clientRepo,
	}
}

// CreatePiece orchestrates piece creation:
// 1. Verify the full ownership chain (User → Client → Project → Order).
// 2. Load the PieceDefinition and verify visibility.
// 3. Build the domain entity (which validates its own fields).
// 4. Validate dimensions against the definition schema.
// 5. Persist via repository.
func (s *pieceService) CreatePiece(ctx context.Context, userID uuid.UUID, clientID uuid.UUID, projectID uuid.UUID, params entities.CreatePieceParams) (*entities.Piece, error) {
	// Step 1: Verify full ownership chain
	_, err := verifyOrderOwnership(ctx, s.clientRepo, s.projectRepo, s.orderRepo, params.OrderID, projectID, clientID, userID)
	if err != nil {
		return nil, err
	}

	// Step 2: Load PieceDefinition and verify visibility
	def, err := verifyPieceDefAccess(ctx, s.pieceDefRepo, params.DefinitionID, userID)
	if err != nil {
		return nil, err
	}

	// Step 3: Domain entity validates its own invariants
	piece, err := entities.NewPiece(params)
	if err != nil {
		return nil, err
	}

	// Step 4: Validate dimensions against the definition schema
	if err := piece.ValidateAgainst(def); err != nil {
		return nil, err
	}

	// Step 5: Persist
	if err := s.pieceRepo.Create(ctx, piece); err != nil {
		return nil, err
	}

	return piece, nil
}

// GetPieceByID retrieves a piece by ID, verifying the full ownership chain.
func (s *pieceService) GetPieceByID(ctx context.Context, id uuid.UUID, orderID uuid.UUID, projectID uuid.UUID, clientID uuid.UUID, userID uuid.UUID) (*entities.Piece, error) {
	return verifyPieceOwnership(ctx, s.clientRepo, s.projectRepo, s.orderRepo, s.pieceRepo, id, orderID, projectID, clientID, userID)
}

// ListPiecesByOrderID retrieves a paginated list of pieces belonging to an order.
// Verifies the full ownership chain.
func (s *pieceService) ListPiecesByOrderID(ctx context.Context, orderID uuid.UUID, projectID uuid.UUID, clientID uuid.UUID, userID uuid.UUID, pg pagination.Pagination) (pagination.Result[*entities.Piece], error) {
	_, err := verifyOrderOwnership(ctx, s.clientRepo, s.projectRepo, s.orderRepo, orderID, projectID, clientID, userID)
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
// Verifies the full ownership chain.
func (s *pieceService) UpdatePiece(ctx context.Context, id uuid.UUID, orderID uuid.UUID, projectID uuid.UUID, clientID uuid.UUID, userID uuid.UUID, params entities.UpdatePieceParams) (*entities.Piece, error) {
	piece, err := verifyPieceOwnership(ctx, s.clientRepo, s.projectRepo, s.orderRepo, s.pieceRepo, id, orderID, projectID, clientID, userID)
	if err != nil {
		return nil, err
	}

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

		// Re-validate dimensions against the definition schema
		def, err := s.pieceDefRepo.GetByID(ctx, piece.DefinitionID)
		if err != nil {
			return nil, err
		}
		if err := piece.ValidateAgainst(def); err != nil {
			return nil, err
		}
	}

	piece.UpdatedAt = time.Now()
	if err := s.pieceRepo.Update(ctx, piece); err != nil {
		return nil, err
	}

	return piece, nil
}

// DeletePiece removes a piece, verifying the full ownership chain.
func (s *pieceService) DeletePiece(ctx context.Context, id uuid.UUID, orderID uuid.UUID, projectID uuid.UUID, clientID uuid.UUID, userID uuid.UUID) error {
	piece, err := verifyPieceOwnership(ctx, s.clientRepo, s.projectRepo, s.orderRepo, s.pieceRepo, id, orderID, projectID, clientID, userID)
	if err != nil {
		return err
	}

	return s.pieceRepo.Delete(ctx, piece.ID)
}
