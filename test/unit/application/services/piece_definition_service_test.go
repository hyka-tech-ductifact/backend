package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ductifact/internal/application/services"
	"ductifact/internal/application/usecases"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/domain/repositories"
	"ductifact/test/unit/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPieceDefService is a test helper that creates a PieceDefinitionService
// with no-op file storage and image processor mocks.
func newPieceDefService(repo *mocks.MockPieceDefinitionRepository) usecases.PieceDefinitionService {
	return services.NewPieceDefinitionService(repo, &mocks.MockFileStorage{}, &mocks.MockImageProcessor{}, &mocks.MockPieceRepository{})
}

// =============================================================================
// CreatePieceDefinition
// =============================================================================

func TestCreatePieceDefinition_WithValidData_ReturnsDef(t *testing.T) {
	userID := uuid.New()
	svc := newPieceDefService(&mocks.MockPieceDefinitionRepository{})

	def, err := svc.CreatePieceDefinition(context.Background(), userID, entities.CreatePieceDefParams{
		Name:            "Rectangle",
		DimensionSchema: []string{"Length", "Width"},
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, "Rectangle", def.Name)
	assert.Equal(t, []string{"Length", "Width"}, def.DimensionSchema)
	assert.False(t, def.Predefined)
}

func TestCreatePieceDefinition_WithEmptyName_ReturnsError(t *testing.T) {
	svc := newPieceDefService(&mocks.MockPieceDefinitionRepository{})

	def, err := svc.CreatePieceDefinition(context.Background(), uuid.New(), entities.CreatePieceDefParams{
		DimensionSchema: []string{"Length"},
	}, nil)

	assert.Nil(t, def)
	assert.ErrorIs(t, err, entities.ErrEmptyPieceDefName)
}

func TestCreatePieceDefinition_WithEmptySchema_ReturnsError(t *testing.T) {
	svc := newPieceDefService(&mocks.MockPieceDefinitionRepository{})

	def, err := svc.CreatePieceDefinition(context.Background(), uuid.New(), entities.CreatePieceDefParams{
		Name:            "Empty",
		DimensionSchema: []string{},
	}, nil)

	assert.Nil(t, def)
	assert.ErrorIs(t, err, entities.ErrNoDimensionFields)
}

func TestCreatePieceDefinition_WhenRepoFails_ReturnsError(t *testing.T) {
	repo := &mocks.MockPieceDefinitionRepository{
		CreateFn: func(ctx context.Context, def *entities.PieceDefinition) error {
			return errors.New("db connection lost")
		},
	}
	svc := newPieceDefService(repo)

	def, err := svc.CreatePieceDefinition(context.Background(), uuid.New(), entities.CreatePieceDefParams{
		Name:            "Rect",
		DimensionSchema: []string{"Length"},
	}, nil)

	assert.Nil(t, def)
	assert.EqualError(t, err, "db connection lost")
}

// =============================================================================
// GetPieceDefinitionByID
// =============================================================================

func TestGetPieceDefinitionByID_WithCustomDef_OwnedByUser_ReturnsDef(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	svc := newPieceDefService(pieceDefRepoReturning(def))

	result, err := svc.GetPieceDefinitionByID(context.Background(), def.ID, userID)

	require.NoError(t, err)
	assert.Equal(t, def.Name, result.Name)
}

func TestGetPieceDefinitionByID_WithPredefinedDef_ReturnsDefForAnyUser(t *testing.T) {
	def := newTestPredefinedPieceDef()
	svc := newPieceDefService(pieceDefRepoReturning(def))

	result, err := svc.GetPieceDefinitionByID(context.Background(), def.ID, uuid.New())

	require.NoError(t, err)
	assert.Equal(t, def.Name, result.Name)
	assert.True(t, result.Predefined)
}

func TestGetPieceDefinitionByID_WithCustomDef_NotOwned_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	def := newTestPieceDef(ownerID)
	svc := newPieceDefService(pieceDefRepoReturning(def))

	result, err := svc.GetPieceDefinitionByID(context.Background(), def.ID, uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, repositories.ErrPieceDefNotOwned)
}

func TestGetPieceDefinitionByID_NotFound_ReturnsError(t *testing.T) {
	svc := newPieceDefService(pieceDefRepoReturning(nil))

	result, err := svc.GetPieceDefinitionByID(context.Background(), uuid.New(), uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, repositories.ErrPieceDefNotFound)
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
		ListByUserIDFn: func(ctx context.Context, uID uuid.UUID, includeArchived bool, pg pagination.Pagination) ([]*entities.PieceDefinition, int64, error) {
			return expected, 2, nil
		},
	}
	svc := newPieceDefService(repo)

	pg, _ := pagination.NewPagination(1, 20)
	result, err := svc.ListPieceDefinitions(context.Background(), userID, false, pg)

	require.NoError(t, err)
	assert.Len(t, result.Data, 2)
	assert.Equal(t, int64(2), result.TotalItems)
}

func TestListPieceDefinitions_EmptyList(t *testing.T) {
	repo := &mocks.MockPieceDefinitionRepository{
		ListByUserIDFn: func(ctx context.Context, uID uuid.UUID, includeArchived bool, pg pagination.Pagination) ([]*entities.PieceDefinition, int64, error) {
			return []*entities.PieceDefinition{}, 0, nil
		},
	}
	svc := newPieceDefService(repo)

	pg, _ := pagination.NewPagination(1, 20)
	result, err := svc.ListPieceDefinitions(context.Background(), uuid.New(), false, pg)

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
	svc := newPieceDefService(pieceDefRepoReturning(def))

	result, err := svc.UpdatePieceDefinition(context.Background(), def.ID, userID, entities.UpdatePieceDefParams{
		Name: strPtr("Updated Name"),
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, "Updated Name", result.Name)
}

func TestUpdatePieceDefinition_WithNewSchema_Updates(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	svc := newPieceDefService(pieceDefRepoReturning(def))

	newSchema := []string{"Height", "Radius"}
	result, err := svc.UpdatePieceDefinition(context.Background(), def.ID, userID, entities.UpdatePieceDefParams{
		DimensionSchema: &newSchema,
	}, nil)

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
	svc := newPieceDefService(repo)

	_, err := svc.UpdatePieceDefinition(context.Background(), def.ID, userID, entities.UpdatePieceDefParams{}, nil)

	require.NoError(t, err)
	assert.False(t, updateCalled, "Update should not be called when no fields change")
}

func TestUpdatePieceDefinition_WithPredefinedDef_ReturnsForbidden(t *testing.T) {
	def := newTestPredefinedPieceDef()
	svc := newPieceDefService(pieceDefRepoReturning(def))

	result, err := svc.UpdatePieceDefinition(context.Background(), def.ID, uuid.New(), entities.UpdatePieceDefParams{
		Name: strPtr("Hacked"),
	}, nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrPieceDefPredefined)
}

func TestUpdatePieceDefinition_NotOwned_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	def := newTestPieceDef(ownerID)
	svc := newPieceDefService(pieceDefRepoReturning(def))

	result, err := svc.UpdatePieceDefinition(context.Background(), def.ID, uuid.New(), entities.UpdatePieceDefParams{
		Name: strPtr("Stolen"),
	}, nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, repositories.ErrPieceDefNotOwned)
}

func TestUpdatePieceDefinition_InUse_ReturnsConflict(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	pieceRepo := &mocks.MockPieceRepository{
		CountByDefinitionIDFn: func(ctx context.Context, definitionID uuid.UUID) (int64, error) {
			return 5, nil
		},
	}
	svc := services.NewPieceDefinitionService(pieceDefRepoReturning(def), &mocks.MockFileStorage{}, &mocks.MockImageProcessor{}, pieceRepo)

	result, err := svc.UpdatePieceDefinition(context.Background(), def.ID, userID, entities.UpdatePieceDefParams{
		Name: strPtr("New Name"),
	}, nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrPieceDefInUse)
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
	svc := newPieceDefService(repo)

	err := svc.DeletePieceDefinition(context.Background(), def.ID, userID)

	assert.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestDeletePieceDefinition_WithPredefinedDef_ReturnsForbidden(t *testing.T) {
	def := newTestPredefinedPieceDef()
	svc := newPieceDefService(pieceDefRepoReturning(def))

	err := svc.DeletePieceDefinition(context.Background(), def.ID, uuid.New())

	assert.ErrorIs(t, err, services.ErrPieceDefPredefined)
}

func TestDeletePieceDefinition_NotOwned_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	def := newTestPieceDef(ownerID)
	svc := newPieceDefService(pieceDefRepoReturning(def))

	err := svc.DeletePieceDefinition(context.Background(), def.ID, uuid.New())

	assert.ErrorIs(t, err, repositories.ErrPieceDefNotOwned)
}

func TestDeletePieceDefinition_NotFound_ReturnsError(t *testing.T) {
	svc := newPieceDefService(pieceDefRepoReturning(nil))

	err := svc.DeletePieceDefinition(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, repositories.ErrPieceDefNotFound)
}

func TestDeletePieceDefinition_InUse_ReturnsConflict(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	pieceRepo := &mocks.MockPieceRepository{
		CountByDefinitionIDFn: func(ctx context.Context, definitionID uuid.UUID) (int64, error) {
			return 15, nil
		},
	}
	repo := pieceDefRepoReturning(def)
	svc := services.NewPieceDefinitionService(repo, &mocks.MockFileStorage{}, &mocks.MockImageProcessor{}, pieceRepo)

	err := svc.DeletePieceDefinition(context.Background(), def.ID, userID)

	assert.ErrorIs(t, err, services.ErrPieceDefInUse)
}

// =============================================================================
// ArchivePieceDefinition
// =============================================================================

func TestArchivePieceDefinition_CustomDef_Succeeds(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	archived := false
	repo := pieceDefRepoReturning(def)
	repo.ArchiveFn = func(ctx context.Context, id uuid.UUID) error {
		archived = true
		return nil
	}
	svc := newPieceDefService(repo)

	result, err := svc.ArchivePieceDefinition(context.Background(), def.ID, userID)

	require.NoError(t, err)
	assert.True(t, archived)
	assert.NotNil(t, result.ArchivedAt)
}

func TestArchivePieceDefinition_AlreadyArchived_IsIdempotent(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	now := time.Now()
	def.ArchivedAt = &now // already archived
	repo := pieceDefRepoReturning(def)
	archiveCalled := false
	repo.ArchiveFn = func(ctx context.Context, id uuid.UUID) error {
		archiveCalled = true
		return nil
	}
	svc := newPieceDefService(repo)

	result, err := svc.ArchivePieceDefinition(context.Background(), def.ID, userID)

	require.NoError(t, err)
	assert.False(t, archiveCalled) // should not call repo
	assert.NotNil(t, result.ArchivedAt)
}

func TestArchivePieceDefinition_PredefinedDef_ReturnsError(t *testing.T) {
	def := newTestPredefinedPieceDef()
	repo := pieceDefRepoReturning(def)
	svc := newPieceDefService(repo)

	result, err := svc.ArchivePieceDefinition(context.Background(), def.ID, uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrPieceDefPredefined)
}

func TestArchivePieceDefinition_NotFound_ReturnsError(t *testing.T) {
	svc := newPieceDefService(pieceDefRepoReturning(nil))

	result, err := svc.ArchivePieceDefinition(context.Background(), uuid.New(), uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, repositories.ErrPieceDefNotFound)
}

// =============================================================================
// UnarchivePieceDefinition
// =============================================================================

func TestUnarchivePieceDefinition_ArchivedDef_Succeeds(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	now := time.Now()
	def.ArchivedAt = &now // currently archived
	unarchived := false
	repo := pieceDefRepoReturning(def)
	repo.UnarchiveFn = func(ctx context.Context, id uuid.UUID) error {
		unarchived = true
		return nil
	}
	svc := newPieceDefService(repo)

	result, err := svc.UnarchivePieceDefinition(context.Background(), def.ID, userID)

	require.NoError(t, err)
	assert.True(t, unarchived)
	assert.Nil(t, result.ArchivedAt)
}

func TestUnarchivePieceDefinition_AlreadyActive_IsIdempotent(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID) // ArchivedAt is nil
	unarchiveCalled := false
	repo := pieceDefRepoReturning(def)
	repo.UnarchiveFn = func(ctx context.Context, id uuid.UUID) error {
		unarchiveCalled = true
		return nil
	}
	svc := newPieceDefService(repo)

	result, err := svc.UnarchivePieceDefinition(context.Background(), def.ID, userID)

	require.NoError(t, err)
	assert.False(t, unarchiveCalled)
	assert.Nil(t, result.ArchivedAt)
}

func TestUnarchivePieceDefinition_PredefinedDef_ReturnsError(t *testing.T) {
	def := newTestPredefinedPieceDef()
	repo := pieceDefRepoReturning(def)
	svc := newPieceDefService(repo)

	result, err := svc.UnarchivePieceDefinition(context.Background(), def.ID, uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrPieceDefPredefined)
}

func TestUnarchivePieceDefinition_NotFound_ReturnsError(t *testing.T) {
	svc := newPieceDefService(pieceDefRepoReturning(nil))

	result, err := svc.UnarchivePieceDefinition(context.Background(), uuid.New(), uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, repositories.ErrPieceDefNotFound)
}
