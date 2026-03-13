package entities

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyClientName = errors.New("client name cannot be empty")
)

type Client struct {
	ID        uuid.UUID
	Name      string
	UserID    uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewClient is the only way to create a valid Client.
// It validates all business rules and returns an error if any fail.
// A Client always belongs to a User (identified by userID).
func NewClient(name string, userID uuid.UUID) (*Client, error) {
	if name == "" {
		return nil, ErrEmptyClientName
	}

	now := time.Now()
	return &Client{
		ID:        uuid.New(),
		Name:      name,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
