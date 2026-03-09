package usecases

import (
	"context"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// ClientService is the inbound port for client operations.
// Inbound adapters (HTTP handlers, CLI, etc.) depend on this interface.
type ClientService interface {
	CreateClient(ctx context.Context, name string, userID uuid.UUID) (*entities.Client, error)
	GetClientByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Client, error)
	ListClientsByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Client, error)
	UpdateClient(ctx context.Context, id uuid.UUID, userID uuid.UUID, name *string) (*entities.Client, error)
	DeleteClient(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
