package http

import (
	"net/http"
	"time"

	"event-service/internal/application/ports"
	"event-service/internal/domain/entities"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EventHandler is an inbound adapter that translates HTTP requests
// into calls to the application's inbound port (EventService).
type EventHandler struct {
	eventService ports.EventService
}

// CreateEventRequest is the HTTP-specific request DTO.
type CreateEventRequest struct {
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description"`
	Location    string    `json:"location" binding:"required"`
	StartTime   time.Time `json:"start_time" binding:"required"`
	EndTime     time.Time `json:"end_time" binding:"required"`
	OrganizerID uuid.UUID `json:"organizer_id" binding:"required"`
}

// EventResponse is the HTTP-specific response DTO.
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

func NewEventHandler(eventService ports.EventService) *EventHandler {
	return &EventHandler{
		eventService: eventService,
	}
}

func (h *EventHandler) CreateEvent(c *gin.Context) {
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Translate HTTP DTO → Domain Entity
	event := entities.NewEvent(
		req.Title,
		req.Description,
		req.Location,
		req.StartTime,
		req.EndTime,
		req.OrganizerID,
	)

	created, err := h.eventService.CreateEvent(c.Request.Context(), event)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Translate Domain Entity → HTTP DTO
	c.JSON(http.StatusCreated, toEventResponse(created))
}

func (h *EventHandler) GetEvent(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event ID"})
		return
	}

	event, err := h.eventService.GetEventByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	c.JSON(http.StatusOK, toEventResponse(event))
}

func toEventResponse(event *entities.Event) *EventResponse {
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
