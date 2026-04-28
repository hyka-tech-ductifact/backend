package usecases

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// ClientService is the inbound port for client operations.
// Inbound adapters (HTTP handlers, CLI, etc.) depend on this interface.
type ClientService interface {
	CreateClient(ctx context.Context, params entities.CreateClientParams) (*entities.Client, error)
	GetClientByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Client, error)
	ListClientsByUserID(
		ctx context.Context,
		userID uuid.UUID,
		pg pagination.Pagination,
	) (pagination.Result[*entities.Client], error)
	UpdateClient(
		ctx context.Context,
		id uuid.UUID,
		userID uuid.UUID,
		params entities.UpdateClientParams,
	) (*entities.Client, error)
	DeleteClient(ctx context.Context, id uuid.UUID, userID uuid.UUID, cascade bool) error
}
