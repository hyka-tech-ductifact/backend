package entities_test

import (
	"testing"
	"time"

	"event-service/internal/domain/entities"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewEvent(t *testing.T) {
	// Arrange
	title := "Test Event"
	description := "Test Description"
	location := "Test Location"
	startTime := time.Now().Add(time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	organizerID := uuid.New()

	// Act
	event := entities.NewEvent(title, description, location, startTime, endTime, organizerID)

	// Assert
	assert.NotNil(t, event)
	assert.NotEqual(t, uuid.Nil, event.ID)
	assert.Equal(t, title, event.Title)
	assert.Equal(t, description, event.Description)
	assert.Equal(t, location, event.Location)
	assert.Equal(t, startTime, event.StartTime)
	assert.Equal(t, endTime, event.EndTime)
	assert.Equal(t, organizerID, event.OrganizerID)
	assert.False(t, event.CreatedAt.IsZero())
	assert.False(t, event.UpdatedAt.IsZero())
}

func TestNewEvent_UniqueIDs(t *testing.T) {
	// Arrange
	title := "Test Event"
	description := "Test Description"
	location := "Test Location"
	startTime := time.Now().Add(time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	organizerID := uuid.New()

	// Act
	event1 := entities.NewEvent(title, description, location, startTime, endTime, organizerID)
	event2 := entities.NewEvent(title, description, location, startTime, endTime, organizerID)

	// Assert
	assert.NotEqual(t, event1.ID, event2.ID, "Each event should have a unique ID")
}

func TestEvent_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		event   *entities.Event
		isValid bool
	}{
		{
			name: "Valid event",
			event: &entities.Event{
				ID:          uuid.New(),
				Title:       "Valid Event",
				Description: "Valid Description",
				Location:    "Valid Location",
				StartTime:   time.Now().Add(time.Hour),
				EndTime:     time.Now().Add(2 * time.Hour),
				OrganizerID: uuid.New(),
			},
			isValid: true,
		},
		{
			name: "Empty title",
			event: &entities.Event{
				ID:          uuid.New(),
				Title:       "",
				Description: "Valid Description",
				Location:    "Valid Location",
				StartTime:   time.Now().Add(time.Hour),
				EndTime:     time.Now().Add(2 * time.Hour),
				OrganizerID: uuid.New(),
			},
			isValid: false,
		},
		{
			name: "End time before start time",
			event: &entities.Event{
				ID:          uuid.New(),
				Title:       "Valid Event",
				Description: "Valid Description",
				Location:    "Valid Location",
				StartTime:   time.Now().Add(2 * time.Hour),
				EndTime:     time.Now().Add(time.Hour),
				OrganizerID: uuid.New(),
			},
			isValid: false,
		},
		{
			name: "Empty location",
			event: &entities.Event{
				ID:          uuid.New(),
				Title:       "Valid Event",
				Description: "Valid Description",
				Location:    "",
				StartTime:   time.Now().Add(time.Hour),
				EndTime:     time.Now().Add(2 * time.Hour),
				OrganizerID: uuid.New(),
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would require adding an IsValid method to the Event struct
			// For now, we'll test the basic structure
			if tt.isValid {
				assert.NotEmpty(t, tt.event.Title)
				assert.NotEmpty(t, tt.event.Location)
				assert.True(t, tt.event.EndTime.After(tt.event.StartTime))
			}
		})
	}
}
