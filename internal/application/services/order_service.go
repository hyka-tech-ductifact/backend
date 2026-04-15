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
	ErrOrderNotFound = errors.New("order not found")
	ErrOrderNotOwned = errors.New("order does not belong to this project")
)

// orderService implements usecases.OrderService.
// Unexported struct: can only be created via NewOrderService.
type orderService struct {
	orderRepo   repositories.OrderRepository
	projectRepo repositories.ProjectRepository
	clientRepo  repositories.ClientRepository
}

// NewOrderService creates a new OrderService.
// It receives the order, project, and client repositories (outbound ports).
// The project and client repositories are needed to verify the full ownership chain:
// User → Client → Project → Order.
func NewOrderService(
	orderRepo repositories.OrderRepository,
	projectRepo repositories.ProjectRepository,
	clientRepo repositories.ClientRepository,
) *orderService {
	return &orderService{
		orderRepo:   orderRepo,
		projectRepo: projectRepo,
		clientRepo:  clientRepo,
	}
}

// CreateOrder orchestrates order creation:
// 1. Verify the full ownership chain (User → Client → Project).
// 2. Build the domain entity (which validates all fields).
// 3. Persist via repository.
func (s *orderService) CreateOrder(ctx context.Context, userID uuid.UUID, clientID uuid.UUID, params entities.CreateOrderParams) (*entities.Order, error) {
	// Step 1: Verify full ownership chain
	_, err := verifyProjectOwnership(ctx, s.clientRepo, s.projectRepo, params.ProjectID, clientID, userID)
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

// GetOrderByID retrieves an order by ID, verifying the full ownership chain.
func (s *orderService) GetOrderByID(ctx context.Context, id uuid.UUID, projectID uuid.UUID, clientID uuid.UUID, userID uuid.UUID) (*entities.Order, error) {
	return verifyOrderOwnership(ctx, s.clientRepo, s.projectRepo, s.orderRepo, id, projectID, clientID, userID)
}

// ListOrdersByProjectID retrieves a paginated list of orders belonging to a project.
// Verifies the full ownership chain.
func (s *orderService) ListOrdersByProjectID(ctx context.Context, projectID uuid.UUID, clientID uuid.UUID, userID uuid.UUID, pg pagination.Pagination) (pagination.Result[*entities.Order], error) {
	_, err := verifyProjectOwnership(ctx, s.clientRepo, s.projectRepo, projectID, clientID, userID)
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
// Verifies the full ownership chain.
func (s *orderService) UpdateOrder(ctx context.Context, id uuid.UUID, projectID uuid.UUID, clientID uuid.UUID, userID uuid.UUID, params entities.UpdateOrderParams) (*entities.Order, error) {
	order, err := verifyOrderOwnership(ctx, s.clientRepo, s.projectRepo, s.orderRepo, id, projectID, clientID, userID)
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

// DeleteOrder removes an order, verifying the full ownership chain.
func (s *orderService) DeleteOrder(ctx context.Context, id uuid.UUID, projectID uuid.UUID, clientID uuid.UUID, userID uuid.UUID) error {
	order, err := verifyOrderOwnership(ctx, s.clientRepo, s.projectRepo, s.orderRepo, id, projectID, clientID, userID)
	if err != nil {
		return err
	}

	return s.orderRepo.Delete(ctx, order.ID)
}
