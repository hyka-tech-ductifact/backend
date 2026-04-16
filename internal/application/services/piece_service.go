package services

import (
	"context"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
)

type pieceService struct {
	pieceRepo    repositories.PieceRepository
	pieceDefRepo repositories.PieceDefinitionRepository
	orderRepo    repositories.OrderRepository
}

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

func (s *pieceService) CreatePiece(ctx context.Context, userID uuid.UUID, params entities.CreatePieceParams) (*entities.Piece, error) {
	_, err := s.orderRepo.GetByIDForOwner(ctx, params.OrderID, userID)
	if err != nil {
		return nil, err
	}

	def, err := s.pieceDefRepo.GetByIDForOwner(ctx, params.DefinitionID, userID)
	if err != nil {
		return nil, err
	}

	piece, err := entities.NewPiece(params)
	if err != nil {
		return nil, err
	}

	if err := piece.ValidateAgainst(def); err != nil {
		return nil, err
	}

	if err := s.pieceRepo.Create(ctx, piece); err != nil {
		return nil, err
	}

	return piece, nil
}

func (s *pieceService) GetPieceByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Piece, error) {
	return s.pieceRepo.GetByIDForOwner(ctx, id, userID)
}

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

func (s *pieceService) UpdatePiece(ctx context.Context, id uuid.UUID, userID uuid.UUID, params entities.UpdatePieceParams) (*entities.Piece, error) {
	piece, err := s.pieceRepo.GetByIDForOwner(ctx, id, userID)
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

func (s *pieceService) DeletePiece(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	piece, err := s.pieceRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return err
	}

	return s.pieceRepo.Delete(ctx, piece.ID)
}
