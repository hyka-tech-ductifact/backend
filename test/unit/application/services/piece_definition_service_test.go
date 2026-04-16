package services_test

import (
	"context"
	"errors"
	"testing"

	"ductifact/internal/application/services"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/test/unit/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CreatePieceDefinition
// =============================================================================

func TestCreatePieceDefinition_WithValidData_ReturnsDef(t *testing.T) {
	userID := uuid.New()
	svc := services.NewPieceDefinitionService(&mocks.MockPieceDefinitionRepository{})

	def, err := svc.CreatePieceDefinition(context.Background(), userID, entities.CreatePieceDefParams{
		Name:            "Rectangle",
		ImageURL:        "https://example.com/rect.png",
		DimensionSchema: []string{"Length", "Width"},
	})

	require.NoError(t, err)
	assert.Equal(t, "Rectangle", def.Name)
	assert.Equal(t, "https://example.com/rect.png", def.ImageURL)
	assert.Equal(t, []string{"Length", "Width"}, def.DimensionSchema)
	assert.False(t, def.Predefined)
}

func TestCreatePieceDefinition_WithEmptyName_ReturnsError(t *testing.T) {
	svc := services.NewPieceDefinitionService(&mocks.MockPieceDefinitionRepository{})

	def, err := svc.CreatePieceDefinition(context.Background(), uuid.New(), entities.CreatePieceDefParams{
		DimensionSchema: []string{"Length"},
	})

	assert.Nil(t, def)
	assert.ErrorIs(t, err, entities.ErrEmptyPieceDefName)
}

func TestCreatePieceDefinition_WithEmptySchema_ReturnsError(t *testing.T) {
	svc := services.NewPieceDefinitionService(&mocks.MockPieceDefinitionRepository{})

	def, err := svc.CreatePieceDefinition(context.Background(), uuid.New(), entities.CreatePieceDefParams{
		Name:            "Empty",
		DimensionSchema: []string{},
	})

	assert.Nil(t, def)
	assert.ErrorIs(t, err, entities.ErrNoDimensionFields)
}

func TestCreatePieceDefinition_WhenRepoFails_ReturnsError(t *testing.T) {
	repo := &mocks.MockPieceDefinitionRepository{
		CreateFn: func(ctx context.Context, def *entities.PieceDefinition) error {
			return errors.New("db connection lost")
		},
	}
	svc := services.NewPieceDefinitionService(repo)

	def, err := svc.CreatePieceDefinition(context.Background(), uuid.New(), entities.CreatePieceDefParams{
		Name:            "Rect",
		DimensionSchema: []string{"Length"},
	})

	assert.Nil(t, def)
	assert.EqualError(t, err, "db connection lost")
}

// =============================================================================
// GetPieceDefinitionByID
// =============================================================================

func TestGetPieceDefinitionByID_WithCustomDef_OwnedByUser_ReturnsDef(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	svc := services.NewPieceDefinitionService(pieceDefRepoReturning(def))

	result, err := svc.GetPieceDefinitionByID(context.Background(), def.ID, userID)

	require.NoError(t, err)
	assert.Equal(t, def.Name, result.Name)
}

func TestGetPieceDefinitionByID_WithPredefinedDef_ReturnsDefForAnyUser(t *testing.T) {
	def := newTestPredefinedPieceDef()
	svc := services.NewPieceDefinitionService(pieceDefRepoReturning(def))

	result, err := svc.GetPieceDefinitionByID(context.Background(), def.ID, uuid.New())

	require.NoError(t, err)
	assert.Equal(t, def.Name, result.Name)
	assert.True(t, result.Predefined)
}

func TestGetPieceDefinitionByID_WithCustomDef_NotOwned_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	def := newTestPieceDef(ownerID)
	svc := services.NewPieceDefinitionService(pieceDefRepoReturning(def))

	result, err := svc.GetPieceDefinitionByID(context.Background(), def.ID, uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrPieceDefNotOwned)
}

func TestGetPieceDefinitionByID_NotFound_ReturnsError(t *testing.T) {
	svc := services.NewPieceDefinitionService(pieceDefRepoReturning(nil))

	result, err := svc.GetPieceDefinitionByID(context.Background(), uuid.New(), uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrPieceDefNotFound)
}

// =============================================================================
// ListPieceDefinitions
// =============================================================================

func TestListPieceDefinitions_ReturnsDefs(t *testing.T) {
	userID := uuid.New()
	expected := []*entities.PieceDefinition{
		{ID: uuid.New(), Name: "Rect", DimensionSchema: []string{"Length"}},
		{ID: uuid.New(), Name: "Circle", DimensionSchema: []string{"Radius"}},
	}
	repo := &mocks.MockPieceDefinitionRepository{
		ListByUserIDFn: func(ctx context.Context, uID uuid.UUID, pg pagination.Pagination) ([]*entities.PieceDefinition, int64, error) {
			return expected, 2, nil
		},
	}
	svc := services.NewPieceDefinitionService(repo)

	pg, _ := pagination.NewPagination(1, 20)
	result, err := svc.ListPieceDefinitions(context.Background(), userID, pg)

	require.NoError(t, err)
	assert.Len(t, result.Data, 2)
	assert.Equal(t, int64(2), result.TotalItems)
}

func TestListPieceDefinitions_EmptyList(t *testing.T) {
	repo := &mocks.MockPieceDefinitionRepository{
		ListByUserIDFn: func(ctx context.Context, uID uuid.UUID, pg pagination.Pagination) ([]*entities.PieceDefinition, int64, error) {
			return []*entities.PieceDefinition{}, 0, nil
		},
	}
	svc := services.NewPieceDefinitionService(repo)

	pg, _ := pagination.NewPagination(1, 20)
	result, err := svc.ListPieceDefinitions(context.Background(), uuid.New(), pg)

	require.NoError(t, err)
	assert.Empty(t, result.Data)
	assert.Equal(t, int64(0), result.TotalItems)
}

// =============================================================================
// UpdatePieceDefinition
// =============================================================================

func TestUpdatePieceDefinition_WithNewName_Updates(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	svc := services.NewPieceDefinitionService(pieceDefRepoReturning(def))

	result, err := svc.UpdatePieceDefinition(context.Background(), def.ID, userID, entities.UpdatePieceDefParams{
		Name: strPtr("Updated Name"),
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Name", result.Name)
}

func TestUpdatePieceDefinition_WithNewSchema_Updates(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	svc := services.NewPieceDefinitionService(pieceDefRepoReturning(def))

	newSchema := []string{"Height", "Radius"}
	result, err := svc.UpdatePieceDefinition(context.Background(), def.ID, userID, entities.UpdatePieceDefParams{
		DimensionSchema: &newSchema,
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"Height", "Radius"}, result.DimensionSchema)
}

func TestUpdatePieceDefinition_WithNoChanges_SkipsPersistence(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	repo := pieceDefRepoReturning(def)
	updateCalled := false
	repo.UpdateFn = func(ctx context.Context, d *entities.PieceDefinition) error {
		updateCalled = true
		return nil
	}
	svc := services.NewPieceDefinitionService(repo)

	_, err := svc.UpdatePieceDefinition(context.Background(), def.ID, userID, entities.UpdatePieceDefParams{})

	require.NoError(t, err)
	assert.False(t, updateCalled, "Update should not be called when no fields change")
}

func TestUpdatePieceDefinition_WithPredefinedDef_ReturnsForbidden(t *testing.T) {
	def := newTestPredefinedPieceDef()
	svc := services.NewPieceDefinitionService(pieceDefRepoReturning(def))

	result, err := svc.UpdatePieceDefinition(context.Background(), def.ID, uuid.New(), entities.UpdatePieceDefParams{
		Name: strPtr("Hacked"),
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrPieceDefPredefined)
}

func TestUpdatePieceDefinition_NotOwned_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	def := newTestPieceDef(ownerID)
	svc := services.NewPieceDefinitionService(pieceDefRepoReturning(def))

	result, err := svc.UpdatePieceDefinition(context.Background(), def.ID, uuid.New(), entities.UpdatePieceDefParams{
		Name: strPtr("Stolen"),
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrPieceDefNotOwned)
}

// =============================================================================
// DeletePieceDefinition
// =============================================================================

func TestDeletePieceDefinition_WithOwnedDef_Succeeds(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	deleteCalled := false
	repo := pieceDefRepoReturning(def)
	repo.DeleteFn = func(ctx context.Context, id uuid.UUID) error {
		deleteCalled = true
		return nil
	}
	svc := services.NewPieceDefinitionService(repo)

	err := svc.DeletePieceDefinition(context.Background(), def.ID, userID)

	assert.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestDeletePieceDefinition_WithPredefinedDef_ReturnsForbidden(t *testing.T) {
	def := newTestPredefinedPieceDef()
	svc := services.NewPieceDefinitionService(pieceDefRepoReturning(def))

	err := svc.DeletePieceDefinition(context.Background(), def.ID, uuid.New())

	assert.ErrorIs(t, err, services.ErrPieceDefPredefined)
}

func TestDeletePieceDefinition_NotOwned_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	def := newTestPieceDef(ownerID)
	svc := services.NewPieceDefinitionService(pieceDefRepoReturning(def))

	err := svc.DeletePieceDefinition(context.Background(), def.ID, uuid.New())

	assert.ErrorIs(t, err, services.ErrPieceDefNotOwned)
}

func TestDeletePieceDefinition_NotFound_ReturnsError(t *testing.T) {
	svc := services.NewPieceDefinitionService(pieceDefRepoReturning(nil))

	err := svc.DeletePieceDefinition(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, services.ErrPieceDefNotFound)
}
