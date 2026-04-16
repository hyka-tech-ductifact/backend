package entities_test

import (
	"testing"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewPiece
// =============================================================================

func TestNewPiece_WithValidData_ReturnsPiece(t *testing.T) {
	orderID := uuid.New()
	defID := uuid.New()
	piece, err := entities.NewPiece(entities.CreatePieceParams{
		Title:        "Side panel",
		OrderID:      orderID,
		DefinitionID: defID,
		Dimensions:   map[string]float64{"Length": 150.5, "Width": 80.0},
		Quantity:     15,
	})

	require.NoError(t, err)
	assert.Equal(t, "Side panel", piece.Title)
	assert.Equal(t, orderID, piece.OrderID)
	assert.Equal(t, defID, piece.DefinitionID)
	assert.Equal(t, map[string]float64{"Length": 150.5, "Width": 80.0}, piece.Dimensions)
	assert.Equal(t, 15, piece.Quantity)
	assert.NotEmpty(t, piece.ID)
	assert.False(t, piece.CreatedAt.IsZero())
	assert.False(t, piece.UpdatedAt.IsZero())
	assert.Nil(t, piece.DeletedAt)
}

func TestNewPiece_WithEmptyTitle_ReturnsError(t *testing.T) {
	piece, err := entities.NewPiece(entities.CreatePieceParams{
		OrderID:      uuid.New(),
		DefinitionID: uuid.New(),
		Dimensions:   map[string]float64{"Length": 10},
		Quantity:     1,
	})

	assert.Nil(t, piece)
	assert.ErrorIs(t, err, entities.ErrEmptyPieceTitle)
}

func TestNewPiece_WithNilOrderID_ReturnsError(t *testing.T) {
	piece, err := entities.NewPiece(entities.CreatePieceParams{
		Title:        "Panel",
		DefinitionID: uuid.New(),
		Dimensions:   map[string]float64{"Length": 10},
		Quantity:     1,
	})

	assert.Nil(t, piece)
	assert.ErrorIs(t, err, entities.ErrNilPieceOrder)
}

func TestNewPiece_WithNilDefinitionID_ReturnsError(t *testing.T) {
	piece, err := entities.NewPiece(entities.CreatePieceParams{
		Title:      "Panel",
		OrderID:    uuid.New(),
		Dimensions: map[string]float64{"Length": 10},
		Quantity:   1,
	})

	assert.Nil(t, piece)
	assert.ErrorIs(t, err, entities.ErrNilPieceDefinition)
}

func TestNewPiece_WithZeroQuantity_ReturnsError(t *testing.T) {
	piece, err := entities.NewPiece(entities.CreatePieceParams{
		Title:        "Panel",
		OrderID:      uuid.New(),
		DefinitionID: uuid.New(),
		Dimensions:   map[string]float64{"Length": 10},
		Quantity:     0,
	})

	assert.Nil(t, piece)
	assert.ErrorIs(t, err, entities.ErrInvalidPieceQuantity)
}

func TestNewPiece_WithNegativeQuantity_ReturnsError(t *testing.T) {
	piece, err := entities.NewPiece(entities.CreatePieceParams{
		Title:        "Panel",
		OrderID:      uuid.New(),
		DefinitionID: uuid.New(),
		Dimensions:   map[string]float64{"Length": 10},
		Quantity:     -5,
	})

	assert.Nil(t, piece)
	assert.ErrorIs(t, err, entities.ErrInvalidPieceQuantity)
}

func TestNewPiece_GeneratesUniqueIDs(t *testing.T) {
	orderID := uuid.New()
	defID := uuid.New()
	params := entities.CreatePieceParams{
		Title: "Panel", OrderID: orderID, DefinitionID: defID,
		Dimensions: map[string]float64{"L": 1}, Quantity: 1,
	}
	p1, err1 := entities.NewPiece(params)
	p2, err2 := entities.NewPiece(params)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, p1.ID, p2.ID, "each piece must have a unique ID")
}

func TestNewPiece_SetsCreatedAtAndUpdatedAtEqual(t *testing.T) {
	piece, err := entities.NewPiece(entities.CreatePieceParams{
		Title: "Panel", OrderID: uuid.New(), DefinitionID: uuid.New(),
		Dimensions: map[string]float64{"L": 1}, Quantity: 1,
	})

	require.NoError(t, err)
	assert.Equal(t, piece.CreatedAt, piece.UpdatedAt,
		"CreatedAt and UpdatedAt must be equal on creation")
}

// =============================================================================
// UpdatePieceParams
// =============================================================================

func TestUpdatePieceParams_HasChanges_WithNoFields_ReturnsFalse(t *testing.T) {
	params := entities.UpdatePieceParams{}
	assert.False(t, params.HasChanges())
}

func TestUpdatePieceParams_HasChanges_WithAnyField_ReturnsTrue(t *testing.T) {
	title := "New"
	dims := map[string]float64{"L": 1}
	qty := 5

	tests := []struct {
		label  string
		params entities.UpdatePieceParams
	}{
		{"Title", entities.UpdatePieceParams{Title: &title}},
		{"Dimensions", entities.UpdatePieceParams{Dimensions: &dims}},
		{"Quantity", entities.UpdatePieceParams{Quantity: &qty}},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			assert.True(t, tc.params.HasChanges())
		})
	}
}

// =============================================================================
// Setter tests
// =============================================================================

func newTestPieceForSetters() *entities.Piece {
	p, _ := entities.NewPiece(entities.CreatePieceParams{
		Title: "Test Piece", OrderID: uuid.New(), DefinitionID: uuid.New(),
		Dimensions: map[string]float64{"Length": 100}, Quantity: 10,
	})
	return p
}

func TestPieceSetTitle_WithValidTitle_Updates(t *testing.T) {
	piece := newTestPieceForSetters()
	err := piece.SetTitle("New Title")

	assert.NoError(t, err)
	assert.Equal(t, "New Title", piece.Title)
}

func TestPieceSetTitle_WithEmptyTitle_ReturnsError(t *testing.T) {
	piece := newTestPieceForSetters()
	err := piece.SetTitle("")

	assert.ErrorIs(t, err, entities.ErrEmptyPieceTitle)
}

func TestPieceSetQuantity_WithValidQuantity_Updates(t *testing.T) {
	piece := newTestPieceForSetters()
	err := piece.SetQuantity(25)

	assert.NoError(t, err)
	assert.Equal(t, 25, piece.Quantity)
}

func TestPieceSetQuantity_WithZero_ReturnsError(t *testing.T) {
	piece := newTestPieceForSetters()
	err := piece.SetQuantity(0)

	assert.ErrorIs(t, err, entities.ErrInvalidPieceQuantity)
}

func TestPieceSetQuantity_WithNegative_ReturnsError(t *testing.T) {
	piece := newTestPieceForSetters()
	err := piece.SetQuantity(-1)

	assert.ErrorIs(t, err, entities.ErrInvalidPieceQuantity)
}

// =============================================================================
// ValidateAgainst
// =============================================================================

func newTestDefinition(schema []string) *entities.PieceDefinition {
	userID := uuid.New()
	return &entities.PieceDefinition{
		ID:              uuid.New(),
		Name:            "Test Def",
		DimensionSchema: schema,
		Predefined:      false,
		UserID:          &userID,
	}
}

func TestValidateAgainst_WithMatchingDimensions_Succeeds(t *testing.T) {
	def := newTestDefinition([]string{"Length", "Width"})
	piece := newTestPieceForSetters()
	piece.Dimensions = map[string]float64{"Length": 100, "Width": 50}

	err := piece.ValidateAgainst(def)
	assert.NoError(t, err)
}

func TestValidateAgainst_WithUnexpectedDimensions_ReturnsError(t *testing.T) {
	def := newTestDefinition([]string{"Length"})
	piece := newTestPieceForSetters()
	piece.Dimensions = map[string]float64{"Length": 100, "Height": 50}

	err := piece.ValidateAgainst(def)
	assert.ErrorIs(t, err, entities.ErrUnexpectedDimensions)
	assert.Contains(t, err.Error(), "Height")
}

func TestValidateAgainst_WithMultipleUnexpectedDimensions_ReturnsAll(t *testing.T) {
	def := newTestDefinition([]string{"Length"})
	piece := newTestPieceForSetters()
	piece.Dimensions = map[string]float64{"Length": 100, "Height": 50, "Depth": 25}

	err := piece.ValidateAgainst(def)
	assert.ErrorIs(t, err, entities.ErrUnexpectedDimensions)
	assert.Contains(t, err.Error(), "Height")
	assert.Contains(t, err.Error(), "Depth")
}

func TestValidateAgainst_WithMissingDimensions_ReturnsError(t *testing.T) {
	def := newTestDefinition([]string{"Length", "Width"})
	piece := newTestPieceForSetters()
	piece.Dimensions = map[string]float64{"Length": 100}

	err := piece.ValidateAgainst(def)
	assert.ErrorIs(t, err, entities.ErrMissingDimensions)
	assert.Contains(t, err.Error(), "Width")
}

func TestValidateAgainst_WithMultipleMissingDimensions_ReturnsAll(t *testing.T) {
	def := newTestDefinition([]string{"Length", "Width", "Height"})
	piece := newTestPieceForSetters()
	piece.Dimensions = map[string]float64{"Length": 100}

	err := piece.ValidateAgainst(def)
	assert.ErrorIs(t, err, entities.ErrMissingDimensions)
	assert.Contains(t, err.Error(), "Width")
	assert.Contains(t, err.Error(), "Height")
}

func TestValidateAgainst_WithZeroDimensionValue_ReturnsError(t *testing.T) {
	def := newTestDefinition([]string{"Length", "Width"})
	piece := newTestPieceForSetters()
	piece.Dimensions = map[string]float64{"Length": 100, "Width": 0}

	err := piece.ValidateAgainst(def)
	assert.ErrorIs(t, err, entities.ErrInvalidDimensionValues)
	assert.Contains(t, err.Error(), "Width")
}

func TestValidateAgainst_WithNegativeDimensionValue_ReturnsError(t *testing.T) {
	def := newTestDefinition([]string{"Length", "Width"})
	piece := newTestPieceForSetters()
	piece.Dimensions = map[string]float64{"Length": 100, "Width": -5}

	err := piece.ValidateAgainst(def)
	assert.ErrorIs(t, err, entities.ErrInvalidDimensionValues)
	assert.Contains(t, err.Error(), "Width")
}

func TestValidateAgainst_WithMultipleInvalidValues_ReturnsAll(t *testing.T) {
	def := newTestDefinition([]string{"Length", "Width"})
	piece := newTestPieceForSetters()
	piece.Dimensions = map[string]float64{"Length": 0, "Width": -5}

	err := piece.ValidateAgainst(def)
	assert.ErrorIs(t, err, entities.ErrInvalidDimensionValues)
	assert.Contains(t, err.Error(), "Length")
	assert.Contains(t, err.Error(), "Width")
}

func TestValidateAgainst_WithEmptyDimensions_ReturnsError(t *testing.T) {
	def := newTestDefinition([]string{"Length"})
	piece := newTestPieceForSetters()
	piece.Dimensions = map[string]float64{}

	err := piece.ValidateAgainst(def)
	assert.ErrorIs(t, err, entities.ErrMissingDimensions)
}
