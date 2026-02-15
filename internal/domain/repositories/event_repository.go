package repositories

import (
	"context"

	"event-service/internal/domain/entities"

	"github.com/google/uuid"
)

type EventRepository interface {
	Create(ctx context.Context, event *entities.Event) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Event, error)
	GetByOrganizerID(ctx context.Context, organizerID uuid.UUID) ([]*entities.Event, error)
	Update(ctx context.Context, event *entities.Event) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*entities.Event, error)
}
