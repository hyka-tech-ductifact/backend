package usecases

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// OrderService is the inbound port for order operations.
// Inbound adapters (HTTP handlers, CLI, etc.) depend on this interface.
type OrderService interface {
	CreateOrder(ctx context.Context, userID uuid.UUID, params entities.CreateOrderParams) (*entities.Order, error)
	GetOrderByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Order, error)
	ListOrdersByProjectID(ctx context.Context, projectID uuid.UUID, userID uuid.UUID, pg pagination.Pagination) (pagination.Result[*entities.Order], error)
	UpdateOrder(ctx context.Context, id uuid.UUID, userID uuid.UUID, params entities.UpdateOrderParams) (*entities.Order, error)
	DeleteOrder(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
