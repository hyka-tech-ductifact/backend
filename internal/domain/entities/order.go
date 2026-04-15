package entities

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyOrderTitle    = errors.New("order title cannot be empty")
	ErrNilOrderProject    = errors.New("order project ID cannot be nil")
	ErrInvalidOrderStatus = errors.New("order status must be 'pending' or 'completed'")
)

// OrderStatus represents the state of an order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusCompleted OrderStatus = "completed"
)

// IsValid returns true if the status is a known value.
func (s OrderStatus) IsValid() bool {
	return s == OrderStatusPending || s == OrderStatusCompleted
}

type Order struct {
	ID          uuid.UUID
	Title       string
	Status      OrderStatus
	Description string
	ProjectID   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time // nil = active, non-nil = soft-deleted
}

// CreateOrderParams groups all parameters needed to create an Order.
// Required: Title, ProjectID. Optional: Status (defaults to "pending"), Description.
type CreateOrderParams struct {
	Title       string
	Status      string
	Description string
	ProjectID   uuid.UUID
}

// UpdateOrderParams groups all fields that can be updated on an Order.
// nil = field not provided (no change), non-nil = new value.
type UpdateOrderParams struct {
	Title       *string
	Status      *string
	Description *string
}

// HasChanges returns true if at least one field is set.
func (p UpdateOrderParams) HasChanges() bool {
	return p.Title != nil || p.Status != nil || p.Description != nil
}

// NewOrder is the only way to create a valid Order.
// It validates all business rules via the setters (single source of truth)
// and returns an error if any fail.
func NewOrder(params CreateOrderParams) (*Order, error) {
	if params.ProjectID == uuid.Nil {
		return nil, ErrNilOrderProject
	}

	now := time.Now()
	o := &Order{
		ID:        uuid.New(),
		ProjectID: params.ProjectID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Validate through setters — each setter is the single source of truth.
	if err := o.SetTitle(params.Title); err != nil {
		return nil, err
	}

	// Default status to "pending" if not provided.
	status := params.Status
	if status == "" {
		status = string(OrderStatusPending)
	}
	if err := o.SetStatus(status); err != nil {
		return nil, err
	}

	o.Description = params.Description

	return o, nil
}

// SetTitle validates and updates the order's title.
func (o *Order) SetTitle(title string) error {
	if title == "" {
		return ErrEmptyOrderTitle
	}
	o.Title = title
	return nil
}

// SetStatus validates and updates the order's status.
func (o *Order) SetStatus(status string) error {
	s := OrderStatus(status)
	if !s.IsValid() {
		return ErrInvalidOrderStatus
	}
	o.Status = s
	return nil
}

// SetDescription updates the order's description. Empty string is valid (clears it).
func (o *Order) SetDescription(description string) {
	o.Description = description
}
