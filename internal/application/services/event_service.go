package services

import (
	"context"
	"errors"

	"event-service/internal/domain/entities"
	"event-service/internal/domain/repositories"

	"github.com/google/uuid"
)

var (
	ErrInvalidEventDuration = errors.New("event end time must be after start time")
)

// eventService implements the inbound port ports.EventService.
// It orchestrates domain entities and outbound ports (repositories).
type eventService struct {
	eventRepo repositories.EventRepository
}

// NewEventService creates a new EventService with the given repository (outbound port).
func NewEventService(eventRepo repositories.EventRepository) *eventService {
	return &eventService{
		eventRepo: eventRepo,
	}
}

func (s *eventService) CreateEvent(ctx context.Context, event *entities.Event) (*entities.Event, error) {
	// Business rule: end time must be after start time
	if event.EndTime.Before(event.StartTime) || event.EndTime.Equal(event.StartTime) {
		return nil, ErrInvalidEventDuration
	}

	err := s.eventRepo.Create(ctx, event)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func (s *eventService) GetEventByID(ctx context.Context, id uuid.UUID) (*entities.Event, error) {
	return s.eventRepo.GetByID(ctx, id)
}
