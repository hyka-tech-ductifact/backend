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

var ErrPieceDefPredefined = errors.New("predefined piece definitions cannot be modified")

// pieceDefinitionService implements usecases.PieceDefinitionService.
// Unexported struct: can only be created via NewPieceDefinitionService.
type pieceDefinitionService struct {
	pieceDefRepo repositories.PieceDefinitionRepository
}

// NewPieceDefinitionService creates a new PieceDefinitionService.
func NewPieceDefinitionService(
	pieceDefRepo repositories.PieceDefinitionRepository,
) *pieceDefinitionService {
	return &pieceDefinitionService{
		pieceDefRepo: pieceDefRepo,
	}
}

// CreatePieceDefinition orchestrates piece definition creation:
// 1. Build the domain entity (which validates all fields).
// 2. Persist via repository.
func (s *pieceDefinitionService) CreatePieceDefinition(ctx context.Context, userID uuid.UUID, params entities.CreatePieceDefParams) (*entities.PieceDefinition, error) {
	// Ensure the creator is set from the authenticated user
	params.UserID = userID

	// Domain entity validates its own invariants
	def, err := entities.NewPieceDefinition(params)
	if err != nil {
		return nil, err
	}

	if err := s.pieceDefRepo.Create(ctx, def); err != nil {
		return nil, err
	}

	return def, nil
}

// GetPieceDefinitionByID retrieves a piece definition by ID.
// Returns the definition only if it is predefined or belongs to the requesting user.
func (s *pieceDefinitionService) GetPieceDefinitionByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.PieceDefinition, error) {
	return s.pieceDefRepo.GetByIDForOwner(ctx, id, userID)
}

// ListPieceDefinitions retrieves a paginated list of piece definitions visible to the user.
// This includes all predefined definitions + the user's custom definitions.
func (s *pieceDefinitionService) ListPieceDefinitions(ctx context.Context, userID uuid.UUID, pg pagination.Pagination) (pagination.Result[*entities.PieceDefinition], error) {
	defs, totalItems, err := s.pieceDefRepo.ListByUserID(ctx, userID, pg)
	if err != nil {
		return pagination.Result[*entities.PieceDefinition]{}, err
	}

	return pagination.NewResult(defs, pg, totalItems), nil
}

// UpdatePieceDefinition applies a partial update to an existing piece definition.
// Only custom (non-predefined) definitions owned by the user can be updated.
func (s *pieceDefinitionService) UpdatePieceDefinition(ctx context.Context, id uuid.UUID, userID uuid.UUID, params entities.UpdatePieceDefParams) (*entities.PieceDefinition, error) {
	def, err := s.pieceDefRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if def.Predefined {
		return nil, ErrPieceDefPredefined
	}

	if !params.HasChanges() {
		return def, nil
	}

	if params.Name != nil {
		if err := def.SetName(*params.Name); err != nil {
			return nil, err
		}
	}
	if params.ImageURL != nil {
		def.SetImageURL(*params.ImageURL)
	}
	if params.DimensionSchema != nil {
		if err := def.SetDimensionSchema(*params.DimensionSchema); err != nil {
			return nil, err
		}
	}

	def.UpdatedAt = time.Now()
	if err := s.pieceDefRepo.Update(ctx, def); err != nil {
		return nil, err
	}

	return def, nil
}

// DeletePieceDefinition removes a piece definition.
// Only custom (non-predefined) definitions owned by the user can be deleted.
func (s *pieceDefinitionService) DeletePieceDefinition(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	def, err := s.pieceDefRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return err
	}

	if def.Predefined {
		return ErrPieceDefPredefined
	}

	return s.pieceDefRepo.Delete(ctx, def.ID)
}
