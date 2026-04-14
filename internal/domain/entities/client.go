package entities

import (
	"ductifact/internal/domain/valueobjects"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyClientName = errors.New("client name cannot be empty")
	ErrNilClientOwner  = errors.New("client owner ID cannot be nil")
)

type Client struct {
	ID          uuid.UUID
	Name        string
	Phone       string
	Email       string
	Description string
	UserID      uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time // nil = active, non-nil = soft-deleted
}

// CreateClientParams groups all parameters needed to create a Client.
// Required: Name, UserID. Optional: Phone, Email, Description.
type CreateClientParams struct {
	Name        string
	Phone       string
	Email       string
	Description string
	UserID      uuid.UUID
}

// UpdateClientParams groups all fields that can be updated on a Client.
// nil = field not provided (no change), non-nil = new value.
type UpdateClientParams struct {
	Name        *string
	Phone       *string
	Email       *string
	Description *string
}

// HasChanges returns true if at least one field is set.
func (p UpdateClientParams) HasChanges() bool {
	return p.Name != nil || p.Phone != nil || p.Email != nil || p.Description != nil
}

// NewClient is the only way to create a valid Client.
// It validates all business rules via the setters (single source of truth)
// and returns an error if any fail.
func NewClient(params CreateClientParams) (*Client, error) {
	if params.UserID == uuid.Nil {
		return nil, ErrNilClientOwner
	}

	now := time.Now()
	c := &Client{
		ID:        uuid.New(),
		UserID:    params.UserID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Validate through setters — each setter is the single source of truth.
	if err := c.SetName(params.Name); err != nil {
		return nil, err
	}
	if err := c.SetPhone(params.Phone); err != nil {
		return nil, err
	}
	if err := c.SetEmail(params.Email); err != nil {
		return nil, err
	}
	if err := c.SetDescription(params.Description); err != nil {
		return nil, err
	}

	return c, nil
}

// SetName validates and updates the client's name.
func (c *Client) SetName(name string) error {
	if name == "" {
		return ErrEmptyClientName
	}
	c.Name = name
	return nil
}

// SetPhone validates format via the Phone VO and updates the client's phone.
func (c *Client) SetPhone(phone string) error {
	valid, err := valueobjects.NewPhone(phone)
	if err != nil {
		return err
	}
	c.Phone = valid.String()
	return nil
}

// SetEmail validates format via the Email VO and updates the client's email.
// An empty email clears the field.
func (c *Client) SetEmail(email string) error {
	if email == "" {
		c.Email = ""
		return nil
	}
	valid, err := valueobjects.NewEmail(email)
	if err != nil {
		return err
	}
	c.Email = valid.String()
	return nil
}

// SetDescription validates via the Description VO and updates the client's description.
func (c *Client) SetDescription(description string) error {
	valid, err := valueobjects.NewDescription(description)
	if err != nil {
		return err
	}
	c.Description = valid.String()
	return nil
}
