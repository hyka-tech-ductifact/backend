package ports

import (
	"context"

	"event-service/internal/domain/entities"

	"github.com/google/uuid"
)

// EventService defines the inbound port for event operations.
// This is the contract that the application exposes to the outside world.
// Inbound adapters (HTTP handlers, gRPC, CLI) depend on this interface.
type EventService interface {
	CreateEvent(ctx context.Context, event *entities.Event) (*entities.Event, error)
	GetEventByID(ctx context.Context, id uuid.UUID) (*entities.Event, error)
}
