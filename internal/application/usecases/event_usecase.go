package usecases

import (
	"context"
	"errors"
	"time"

	"event-service/internal/domain/entities"
	"event-service/internal/domain/repositories"

	"github.com/google/uuid"
)

var (
	ErrInvalidEventDuration = errors.New("event end time must be after start time")
)

type EventUseCase struct {
	eventRepo repositories.EventRepository
}

type CreateEventRequest struct {
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description"`
	Location    string    `json:"location" binding:"required"`
	StartTime   time.Time `json:"start_time" binding:"required"`
	EndTime     time.Time `json:"end_time" binding:"required"`
	OrganizerID uuid.UUID `json:"organizer_id" binding:"required"`
}

type EventResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	OrganizerID uuid.UUID `json:"organizer_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewEventUseCase(eventRepo repositories.EventRepository) *EventUseCase {
	return &EventUseCase{
		eventRepo: eventRepo,
	}
}

func (uc *EventUseCase) CreateEvent(ctx context.Context, req *CreateEventRequest) (*EventResponse, error) {
	// Validate that end time is after start time
	if req.EndTime.Before(req.StartTime) || req.EndTime.Equal(req.StartTime) {
		return nil, ErrInvalidEventDuration
	}

	event := entities.NewEvent(
		req.Title,
		req.Description,
		req.Location,
		req.StartTime,
		req.EndTime,
		req.OrganizerID,
	)

	err := uc.eventRepo.Create(ctx, event)
	if err != nil {
		return nil, err
	}

	return uc.eventToResponse(event), nil
}

func (uc *EventUseCase) GetEventByID(ctx context.Context, id uuid.UUID) (*EventResponse, error) {
	event, err := uc.eventRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return uc.eventToResponse(event), nil
}

func (uc *EventUseCase) eventToResponse(event *entities.Event) *EventResponse {
	return &EventResponse{
		ID:          event.ID,
		Title:       event.Title,
		Description: event.Description,
		Location:    event.Location,
		StartTime:   event.StartTime,
		EndTime:     event.EndTime,
		OrganizerID: event.OrganizerID,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.UpdatedAt,
	}
}
