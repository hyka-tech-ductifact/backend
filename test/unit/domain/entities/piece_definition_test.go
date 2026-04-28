package entities_test

import (
	"testing"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewPieceDefinition
// =============================================================================

func TestNewPieceDefinition_WithValidData_ReturnsPieceDefinition(t *testing.T) {
	userID := uuid.New()
	def, err := entities.NewPieceDefinition(entities.CreatePieceDefParams{
		Name:            "Rectangle",
		ImageURL:        "https://example.com/rect.png",
		DimensionSchema: []string{"Length", "Width"},
		UserID:          userID,
	})

	require.NoError(t, err)
	assert.Equal(t, "Rectangle", def.Name)
	assert.Equal(t, "https://example.com/rect.png", def.ImageURL)
	assert.Equal(t, []string{"Length", "Width"}, def.DimensionSchema)
	assert.False(t, def.Predefined)
	assert.Equal(t, &userID, def.UserID)
	assert.NotEmpty(t, def.ID)
	assert.False(t, def.CreatedAt.IsZero())
	assert.False(t, def.UpdatedAt.IsZero())
	assert.Nil(t, def.DeletedAt)
}

func TestNewPieceDefinition_WithEmptyName_ReturnsError(t *testing.T) {
	def, err := entities.NewPieceDefinition(entities.CreatePieceDefParams{
		DimensionSchema: []string{"Length"},
		UserID:          uuid.New(),
	})

	assert.Nil(t, def)
	assert.ErrorIs(t, err, entities.ErrEmptyPieceDefName)
}

func TestNewPieceDefinition_WithNilUserID_ReturnsError(t *testing.T) {
	def, err := entities.NewPieceDefinition(entities.CreatePieceDefParams{
		Name:            "Rectangle",
		DimensionSchema: []string{"Length"},
	})

	assert.Nil(t, def)
	assert.ErrorIs(t, err, entities.ErrNilClientOwner)
}

func TestNewPieceDefinition_WithEmptySchema_ReturnsError(t *testing.T) {
	def, err := entities.NewPieceDefinition(entities.CreatePieceDefParams{
		Name:            "Empty",
		DimensionSchema: []string{},
		UserID:          uuid.New(),
	})

	assert.Nil(t, def)
	assert.ErrorIs(t, err, entities.ErrNoDimensionFields)
}

func TestNewPieceDefinition_WithTooManyFields_ReturnsError(t *testing.T) {
	schema := make([]string, entities.MaxDimensionFields+1)
	for i := range schema {
		schema[i] = "Dim" + string(rune('A'+i))
	}
	def, err := entities.NewPieceDefinition(entities.CreatePieceDefParams{
		Name:            "TooMany",
		DimensionSchema: schema,
		UserID:          uuid.New(),
	})

	assert.Nil(t, def)
	assert.ErrorIs(t, err, entities.ErrTooManyDimensionFields)
}

func TestNewPieceDefinition_WithDuplicateLabels_ReturnsError(t *testing.T) {
	def, err := entities.NewPieceDefinition(entities.CreatePieceDefParams{
		Name:            "Dupes",
		DimensionSchema: []string{"Length", "Width", "Length"},
		UserID:          uuid.New(),
	})

	assert.Nil(t, def)
	assert.ErrorIs(t, err, entities.ErrDuplicateDimensionLabel)
}

func TestNewPieceDefinition_WithEmptyLabel_ReturnsError(t *testing.T) {
	def, err := entities.NewPieceDefinition(entities.CreatePieceDefParams{
		Name:            "EmptyLabel",
		DimensionSchema: []string{"Length", ""},
		UserID:          uuid.New(),
	})

	assert.Nil(t, def)
	assert.ErrorIs(t, err, entities.ErrEmptyDimensionLabel)
}

func TestNewPieceDefinition_GeneratesUniqueIDs(t *testing.T) {
	userID := uuid.New()
	d1, err1 := entities.NewPieceDefinition(
		entities.CreatePieceDefParams{Name: "A", DimensionSchema: []string{"X"}, UserID: userID},
	)
	d2, err2 := entities.NewPieceDefinition(
		entities.CreatePieceDefParams{Name: "B", DimensionSchema: []string{"Y"}, UserID: userID},
	)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, d1.ID, d2.ID, "each definition must have a unique ID")
}

func TestNewPieceDefinition_SetsCreatedAtAndUpdatedAtEqual(t *testing.T) {
	def, err := entities.NewPieceDefinition(entities.CreatePieceDefParams{
		Name:            "Timestamps",
		DimensionSchema: []string{"Length"},
		UserID:          uuid.New(),
	})

	require.NoError(t, err)
	assert.Equal(t, def.CreatedAt, def.UpdatedAt,
		"CreatedAt and UpdatedAt must be equal on creation")
}

func TestNewPieceDefinition_IsNotPredefined(t *testing.T) {
	def, err := entities.NewPieceDefinition(entities.CreatePieceDefParams{
		Name:            "Custom",
		DimensionSchema: []string{"Length"},
		UserID:          uuid.New(),
	})

	require.NoError(t, err)
	assert.False(t, def.Predefined, "user-created definitions must not be predefined")
}

// =============================================================================
// UpdatePieceDefParams
// =============================================================================

func TestUpdatePieceDefParams_HasChanges_WithNoFields_ReturnsFalse(t *testing.T) {
	params := entities.UpdatePieceDefParams{}
	assert.False(t, params.HasChanges())
}

func TestUpdatePieceDefParams_HasChanges_WithAnyField_ReturnsTrue(t *testing.T) {
	name := "New"
	imageURL := "https://example.com/new.png"
	schema := []string{"A"}

	tests := []struct {
		label  string
		params entities.UpdatePieceDefParams
	}{
		{"Name", entities.UpdatePieceDefParams{Name: &name}},
		{"ImageURL", entities.UpdatePieceDefParams{ImageURL: &imageURL}},
		{"DimensionSchema", entities.UpdatePieceDefParams{DimensionSchema: &schema}},
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

func newTestPieceDefForSetters() *entities.PieceDefinition {
	d, _ := entities.NewPieceDefinition(entities.CreatePieceDefParams{
		Name:            "Test Def",
		DimensionSchema: []string{"Length", "Width"},
		UserID:          uuid.New(),
	})
	return d
}

func TestPieceDefSetName_WithValidName_Updates(t *testing.T) {
	def := newTestPieceDefForSetters()
	err := def.SetName("New Name")

	assert.NoError(t, err)
	assert.Equal(t, "New Name", def.Name)
}

func TestPieceDefSetName_WithEmptyName_ReturnsError(t *testing.T) {
	def := newTestPieceDefForSetters()
	err := def.SetName("")

	assert.ErrorIs(t, err, entities.ErrEmptyPieceDefName)
}

func TestPieceDefSetDimensionSchema_WithValidSchema_Updates(t *testing.T) {
	def := newTestPieceDefForSetters()
	err := def.SetDimensionSchema([]string{"Height", "Radius"})

	assert.NoError(t, err)
	assert.Equal(t, []string{"Height", "Radius"}, def.DimensionSchema)
}

func TestPieceDefSetDimensionSchema_WithEmptySchema_ReturnsError(t *testing.T) {
	def := newTestPieceDefForSetters()
	err := def.SetDimensionSchema([]string{})

	assert.ErrorIs(t, err, entities.ErrNoDimensionFields)
}

func TestPieceDefSetDimensionSchema_WithDuplicates_ReturnsError(t *testing.T) {
	def := newTestPieceDefForSetters()
	err := def.SetDimensionSchema([]string{"A", "B", "A"})

	assert.ErrorIs(t, err, entities.ErrDuplicateDimensionLabel)
}

func TestPieceDefSetImageURL_Updates(t *testing.T) {
	def := newTestPieceDefForSetters()
	def.SetImageURL("https://example.com/updated.png")

	assert.Equal(t, "https://example.com/updated.png", def.ImageURL)
}

func TestPieceDefSetImageURL_WithEmpty_ClearsField(t *testing.T) {
	def := newTestPieceDefForSetters()
	def.SetImageURL("https://example.com/img.png")
	def.SetImageURL("")

	assert.Equal(t, "", def.ImageURL)
}
