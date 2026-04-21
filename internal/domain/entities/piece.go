package entities

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyPieceTitle        = errors.New("piece title cannot be empty")
	ErrNilPieceOrder          = errors.New("piece order ID cannot be nil")
	ErrNilPieceDefinition     = errors.New("piece definition ID cannot be nil")
	ErrInvalidPieceQuantity   = errors.New("piece quantity must be at least 1")
	ErrMissingDimensions      = errors.New("missing required dimensions")
	ErrUnexpectedDimensions   = errors.New("unexpected dimensions")
	ErrInvalidDimensionValues = errors.New("dimension values must be positive")
)

type Piece struct {
	ID           uuid.UUID
	Title        string
	OrderID      uuid.UUID
	DefinitionID uuid.UUID
	Dimensions   map[string]float64
	Quantity     int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time // nil = active, non-nil = soft-deleted
}

// CreatePieceParams groups all parameters needed to create a Piece.
// Required: Title, OrderID, DefinitionID, Dimensions, Quantity.
type CreatePieceParams struct {
	Title        string
	OrderID      uuid.UUID
	DefinitionID uuid.UUID
	Dimensions   map[string]float64
	Quantity     int
}

// UpdatePieceParams groups all fields that can be updated on a Piece.
// nil = field not provided (no change), non-nil = new value.
type UpdatePieceParams struct {
	Title      *string
	Dimensions *map[string]float64
	Quantity   *int
}

// HasChanges returns true if at least one field is set.
func (p UpdatePieceParams) HasChanges() bool {
	return p.Title != nil || p.Dimensions != nil || p.Quantity != nil
}

// NewPiece is the only way to create a valid Piece.
// It validates all business rules via the setters (single source of truth)
// and returns an error if any fail.
func NewPiece(params CreatePieceParams) (*Piece, error) {
	if params.OrderID == uuid.Nil {
		return nil, ErrNilPieceOrder
	}
	if params.DefinitionID == uuid.Nil {
		return nil, ErrNilPieceDefinition
	}

	now := time.Now()
	p := &Piece{
		ID:           uuid.New(),
		OrderID:      params.OrderID,
		DefinitionID: params.DefinitionID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Validate through setters — each setter is the single source of truth.
	if err := p.SetTitle(params.Title); err != nil {
		return nil, err
	}

	if err := p.SetQuantity(params.Quantity); err != nil {
		return nil, err
	}

	// Dimensions are set directly here; schema validation happens in the service layer
	// because it requires loading the PieceDefinition.
	p.Dimensions = params.Dimensions

	return p, nil
}

// SetTitle validates and updates the piece's title.
func (p *Piece) SetTitle(title string) error {
	if title == "" {
		return ErrEmptyPieceTitle
	}
	p.Title = title
	return nil
}

// SetQuantity validates and updates the piece's quantity.
func (p *Piece) SetQuantity(quantity int) error {
	if quantity < 1 {
		return ErrInvalidPieceQuantity
	}
	p.Quantity = quantity
	return nil
}

// ValidateAgainst checks that the piece's dimensions match the definition's schema exactly.
// Every required dimension must be present with a positive value, and no extra dimensions are allowed.
func (p *Piece) ValidateAgainst(def *PieceDefinition) error {
	required := make(map[string]bool, len(def.DimensionSchema))
	for _, label := range def.DimensionSchema {
		required[label] = true
	}

	// Check for unexpected dimensions
	var unexpected []string
	for key := range p.Dimensions {
		if !required[key] {
			unexpected = append(unexpected, key)
		} else {
			delete(required, key)
		}
	}
	if len(unexpected) > 0 {
		return fmt.Errorf("%w: %s", ErrUnexpectedDimensions, strings.Join(unexpected, ", "))
	}

	// Check for missing dimensions
	if len(required) > 0 {
		missing := make([]string, 0, len(required))
		for key := range required {
			missing = append(missing, key)
		}
		return fmt.Errorf("%w: %s", ErrMissingDimensions, strings.Join(missing, ", "))
	}

	// Check all values are positive
	var invalid []string
	for key, val := range p.Dimensions {
		if val <= 0 {
			invalid = append(invalid, fmt.Sprintf("%s = %f", key, val))
		}
	}
	if len(invalid) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidDimensionValues, strings.Join(invalid, ", "))
	}

	return nil
}
