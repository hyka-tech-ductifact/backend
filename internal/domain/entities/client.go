package entities

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyClientName = errors.New("client name cannot be empty")
	ErrNilClientOwner  = errors.New("client owner ID cannot be nil")
)

type Client struct {
	ID        uuid.UUID
	Name      string
	UserID    uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // nil = active, non-nil = soft-deleted
}

// NewClient is the only way to create a valid Client.
// It validates all business rules and returns an error if any fail.
// A Client always belongs to a User (identified by userID).
func NewClient(name string, userID uuid.UUID) (*Client, error) {
	if name == "" {
		return nil, ErrEmptyClientName
	}
	if userID == uuid.Nil {
		return nil, ErrNilClientOwner
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

// SetName validates and updates the client's name.
func (c *Client) SetName(name string) error {
	if name == "" {
		return ErrEmptyClientName
	}
	c.Name = name
	return nil
}
