package services

import (
	"context"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
)

type orderService struct {
	orderRepo   repositories.OrderRepository
	projectRepo repositories.ProjectRepository
}

func NewOrderService(
	orderRepo repositories.OrderRepository,
	projectRepo repositories.ProjectRepository,
) *orderService {
	return &orderService{
		orderRepo:   orderRepo,
		projectRepo: projectRepo,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, userID uuid.UUID, params entities.CreateOrderParams) (*entities.Order, error) {
	_, err := s.projectRepo.GetByIDForOwner(ctx, params.ProjectID, userID)
	if err != nil {
		return nil, err
	}

	order, err := entities.NewOrder(params)
	if err != nil {
		return nil, err
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

func (s *orderService) GetOrderByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Order, error) {
	return s.orderRepo.GetByIDForOwner(ctx, id, userID)
}

func (s *orderService) ListOrdersByProjectID(ctx context.Context, projectID uuid.UUID, userID uuid.UUID, pg pagination.Pagination) (pagination.Result[*entities.Order], error) {
	_, err := s.projectRepo.GetByIDForOwner(ctx, projectID, userID)
	if err != nil {
		return pagination.Result[*entities.Order]{}, err
	}

	orders, totalItems, err := s.orderRepo.ListByProjectID(ctx, projectID, pg)
	if err != nil {
		return pagination.Result[*entities.Order]{}, err
	}

	return pagination.NewResult(orders, pg, totalItems), nil
}

func (s *orderService) UpdateOrder(ctx context.Context, id uuid.UUID, userID uuid.UUID, params entities.UpdateOrderParams) (*entities.Order, error) {
	order, err := s.orderRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if !params.HasChanges() {
		return order, nil
	}

	if params.Title != nil {
		if err := order.SetTitle(*params.Title); err != nil {
			return nil, err
		}
	}
	if params.Status != nil {
		if err := order.SetStatus(*params.Status); err != nil {
			return nil, err
		}
	}
	if params.Description != nil {
		order.SetDescription(*params.Description)
	}

	order.UpdatedAt = time.Now()
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

func (s *orderService) DeleteOrder(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	order, err := s.orderRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return err
	}

	return s.orderRepo.Delete(ctx, order.ID)
}
