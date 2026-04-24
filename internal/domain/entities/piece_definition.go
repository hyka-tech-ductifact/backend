package entities

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyPieceDefName       = errors.New("piece definition name cannot be empty")
	ErrTooManyDimensionFields  = errors.New("piece definition cannot have more than 10 dimension fields")
	ErrNoDimensionFields       = errors.New("piece definition must have at least one dimension field")
	ErrDuplicateDimensionLabel = errors.New("dimension labels must be unique")
	ErrEmptyDimensionLabel     = errors.New("dimension label cannot be empty")
)

const MaxDimensionFields = 10

// DimensionSchema is a list of dimension labels that a piece requires.
// Each label is a human-readable name (e.g. "Length", "Width", "Radius").
// The label also serves as the key in the Piece's Dimensions map.

type PieceDefinition struct {
	ID              uuid.UUID
	Name            string
	ImageURL        string
	DimensionSchema []string
	Predefined      bool
	UserID          *uuid.UUID // nil if predefined
	ArchivedAt      *time.Time // nil = active, non-nil = archived (soft-disabled)
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time // nil = active, non-nil = soft-deleted
}

// IsArchived returns true if the piece definition has been archived.
func (pd *PieceDefinition) IsArchived() bool {
	return pd.ArchivedAt != nil
}

// CreatePieceDefParams groups all parameters needed to create a PieceDefinition.
// Required: Name, DimensionSchema. Optional: ImageURL.
type CreatePieceDefParams struct {
	Name            string
	ImageURL        string
	DimensionSchema []string
	UserID          uuid.UUID
}

// UpdatePieceDefParams groups all fields that can be updated on a PieceDefinition.
// nil = field not provided (no change), non-nil = new value.
type UpdatePieceDefParams struct {
	Name            *string
	ImageURL        *string
	DimensionSchema *[]string
}

// HasChanges returns true if at least one field is set.
func (p UpdatePieceDefParams) HasChanges() bool {
	return p.Name != nil || p.ImageURL != nil || p.DimensionSchema != nil
}

// NewPieceDefinition is the only way to create a valid PieceDefinition.
// It validates all business rules via the setters (single source of truth)
// and returns an error if any fail.
func NewPieceDefinition(params CreatePieceDefParams) (*PieceDefinition, error) {
	if params.UserID == uuid.Nil {
		return nil, ErrNilClientOwner
	}

	now := time.Now()
	userID := params.UserID
	pd := &PieceDefinition{
		ID:         uuid.New(),
		Predefined: false,
		UserID:     &userID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Validate through setters — each setter is the single source of truth.
	if err := pd.SetName(params.Name); err != nil {
		return nil, err
	}

	pd.ImageURL = params.ImageURL

	if err := pd.SetDimensionSchema(params.DimensionSchema); err != nil {
		return nil, err
	}

	return pd, nil
}

// SetName validates and updates the piece definition's name.
func (pd *PieceDefinition) SetName(name string) error {
	if name == "" {
		return ErrEmptyPieceDefName
	}
	pd.Name = name
	return nil
}

// SetDimensionSchema validates and updates the dimension schema.
func (pd *PieceDefinition) SetDimensionSchema(schema []string) error {
	if len(schema) == 0 {
		return ErrNoDimensionFields
	}
	if len(schema) > MaxDimensionFields {
		return ErrTooManyDimensionFields
	}

	seen := make(map[string]bool, len(schema))
	for _, label := range schema {
		if label == "" {
			return ErrEmptyDimensionLabel
		}
		if seen[label] {
			return ErrDuplicateDimensionLabel
		}
		seen[label] = true
	}

	pd.DimensionSchema = schema
	return nil
}

// SetImageURL updates the piece definition's image URL. Empty string clears it.
func (pd *PieceDefinition) SetImageURL(imageURL string) {
	pd.ImageURL = imageURL
}
