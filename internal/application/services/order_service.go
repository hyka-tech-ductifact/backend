package services

import (
	"context"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
)

// orderService implements usecases.OrderService.
// Unexported struct: can only be created via NewOrderService.
type orderService struct {
	orderRepo   repositories.OrderRepository
	projectRepo repositories.ProjectRepository
}

// NewOrderService creates a new OrderService.
// The project repository is needed to verify ownership during Create and List.
func NewOrderService(
	orderRepo repositories.OrderRepository,
	projectRepo repositories.ProjectRepository,
) *orderService {
	return &orderService{
		orderRepo:   orderRepo,
		projectRepo: projectRepo,
	}
}

// CreateOrder orchestrates order creation:
// 1. Verify the owning project belongs to the user.
// 2. Build the domain entity (which validates all fields).
// 3. Persist via repository.
func (s *orderService) CreateOrder(ctx context.Context, userID uuid.UUID, params entities.CreateOrderParams) (*entities.Order, error) {
	// Step 1: Verify project ownership
	_, err := s.projectRepo.GetByIDForOwner(ctx, params.ProjectID, userID)
	if err != nil {
		return nil, err
	}

	// Step 2: Domain entity validates its own invariants
	order, err := entities.NewOrder(params)
	if err != nil {
		return nil, err
	}

	// Step 3: Persist
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

// GetOrderByID retrieves an order by ID, ensuring it belongs to the given user.
func (s *orderService) GetOrderByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Order, error) {
	return s.orderRepo.GetByIDForOwner(ctx, id, userID)
}

// ListOrdersByProjectID retrieves a paginated list of orders for a project,
// ensuring the project belongs to the given user.
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

// UpdateOrder applies a partial update to an existing order.
// Only non-nil fields in params are updated. Ensures the order belongs to the given user.
func (s *orderService) UpdateOrder(ctx context.Context, id uuid.UUID, userID uuid.UUID, params entities.UpdateOrderParams) (*entities.Order, error) {
	order, err := s.orderRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Nothing to update
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

	// Update timestamp and persist
	order.UpdatedAt = time.Now()
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

// DeleteOrder removes an order, ensuring it belongs to the given user.
func (s *orderService) DeleteOrder(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	order, err := s.orderRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return err
	}

	return s.orderRepo.Delete(ctx, order.ID)
}
