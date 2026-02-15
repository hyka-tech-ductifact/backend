package entities

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
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

func NewEvent(title, description, location string, startTime, endTime time.Time, organizerID uuid.UUID) *Event {
	return &Event{
		ID:          uuid.New(),
		Title:       title,
		Description: description,
		Location:    location,
		StartTime:   startTime,
		EndTime:     endTime,
		OrganizerID: organizerID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
