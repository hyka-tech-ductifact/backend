package persistence_test

import (
	"context"
	"testing"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/infrastructure/adapters/outbound/persistence"
	"ductifact/test/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupPieceDefRepo creates piece definition and user repos with a clean DB.
func setupPieceDefRepo(t *testing.T) (
	*persistence.PostgresPieceDefinitionRepository,
	*persistence.PostgresUserRepository,
) {
	db := helpers.SetupTestDB(t)
	helpers.CleanDB(t, db)
	return persistence.NewPostgresPieceDefinitionRepository(db),
		persistence.NewPostgresUserRepository(db)
}

// createTestPieceDef is a helper that creates and persists a custom piece definition.
func createTestPieceDef(t *testing.T, defRepo *persistence.PostgresPieceDefinitionRepository, userID uuid.UUID) *entities.PieceDefinition {
	def, err := entities.NewPieceDefinition(entities.CreatePieceDefParams{
		Name:            "Test Def " + uuid.New().String()[:8],
		DimensionSchema: []string{"Length", "Width"},
		UserID:          userID,
	})
	require.NoError(t, err)
	require.NoError(t, defRepo.Create(context.Background(), def))
	return def
}

// =============================================================================
// Create + GetByID
// =============================================================================

func TestPostgresPieceDefRepository_Create_And_GetByID(t *testing.T) {
	defRepo, userRepo := setupPieceDefRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	def, err := entities.NewPieceDefinition(entities.CreatePieceDefParams{
		Name:            "Rectangle",
		ImageURL:        "https://example.com/rect.png",
		DimensionSchema: []string{"Length", "Width"},
		UserID:          user.ID,
	})
	require.NoError(t, err)

	err = defRepo.Create(ctx, def)
	require.NoError(t, err)

	found, err := defRepo.GetByID(ctx, def.ID)
	require.NoError(t, err)

	assert.Equal(t, def.ID, found.ID)
	assert.Equal(t, "Rectangle", found.Name)
	assert.Equal(t, "https://example.com/rect.png", found.ImageURL)
	assert.Equal(t, []string{"Length", "Width"}, found.DimensionSchema)
	assert.False(t, found.Predefined)
	assert.Equal(t, &user.ID, found.UserID)
	assert.False(t, found.CreatedAt.IsZero())
	assert.False(t, found.UpdatedAt.IsZero())
}

// =============================================================================
// ListByUserID
// =============================================================================

func TestPostgresPieceDefRepository_ListByUserID(t *testing.T) {
	defRepo, userRepo := setupPieceDefRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	createTestPieceDef(t, defRepo, user.ID)
	createTestPieceDef(t, defRepo, user.ID)

	pg, _ := pagination.NewPagination(1, 20)
	defs, total, err := defRepo.ListByUserID(ctx, user.ID, false, pg)
	require.NoError(t, err)
	assert.Len(t, defs, 2)
	assert.Equal(t, int64(2), total)
}

func TestPostgresPieceDefRepository_ListByUserID_Empty(t *testing.T) {
	defRepo, userRepo := setupPieceDefRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	pg, _ := pagination.NewPagination(1, 20)
	defs, total, err := defRepo.ListByUserID(ctx, user.ID, false, pg)
	require.NoError(t, err)
	assert.Empty(t, defs)
	assert.Equal(t, int64(0), total)
}

func TestPostgresPieceDefRepository_ListByUserID_DoesNotReturnOtherUsersCustomDefs(t *testing.T) {
	defRepo, userRepo := setupPieceDefRepo(t)
	ctx := context.Background()

	user1 := createTestUser(t, userRepo)
	user2 := createTestUser(t, userRepo)
	createTestPieceDef(t, defRepo, user1.ID)
	createTestPieceDef(t, defRepo, user2.ID)

	pg, _ := pagination.NewPagination(1, 20)
	defs, total, err := defRepo.ListByUserID(ctx, user1.ID, false, pg)
	require.NoError(t, err)
	assert.Len(t, defs, 1)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, &user1.ID, defs[0].UserID)
}

// =============================================================================
// Update
// =============================================================================

func TestPostgresPieceDefRepository_Update(t *testing.T) {
	defRepo, userRepo := setupPieceDefRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	def := createTestPieceDef(t, defRepo, user.ID)

	require.NoError(t, def.SetName("Updated Name"))
	require.NoError(t, def.SetDimensionSchema([]string{"Height", "Radius"}))
	def.SetImageURL("https://example.com/updated.png")

	err := defRepo.Update(ctx, def)
	require.NoError(t, err)

	found, err := defRepo.GetByID(ctx, def.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", found.Name)
	assert.Equal(t, []string{"Height", "Radius"}, found.DimensionSchema)
	assert.Equal(t, "https://example.com/updated.png", found.ImageURL)
}

// =============================================================================
// Delete
// =============================================================================

func TestPostgresPieceDefRepository_Delete(t *testing.T) {
	defRepo, userRepo := setupPieceDefRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	def := createTestPieceDef(t, defRepo, user.ID)

	err := defRepo.Delete(ctx, def.ID)
	require.NoError(t, err)

	found, err := defRepo.GetByID(ctx, def.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

// =============================================================================
// GetByID — Not Found
// =============================================================================

func TestPostgresPieceDefRepository_GetByID_NotFound(t *testing.T) {
	defRepo, _ := setupPieceDefRepo(t)
	ctx := context.Background()

	found, err := defRepo.GetByID(ctx, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, found)
}

// =============================================================================
// Mapper — Data integrity
// =============================================================================

func TestPostgresPieceDefRepository_Mapper_PreservesAllFields(t *testing.T) {
	defRepo, userRepo := setupPieceDefRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	original, _ := entities.NewPieceDefinition(entities.CreatePieceDefParams{
		Name:            "Full Data Def",
		ImageURL:        "https://example.com/full.png",
		DimensionSchema: []string{"A", "B", "C"},
		UserID:          user.ID,
	})
	require.NoError(t, defRepo.Create(ctx, original))

	found, err := defRepo.GetByID(ctx, original.ID)
	require.NoError(t, err)

	assert.Equal(t, original.ID, found.ID)
	assert.Equal(t, original.Name, found.Name)
	assert.Equal(t, original.ImageURL, found.ImageURL)
	assert.Equal(t, original.DimensionSchema, found.DimensionSchema)
	assert.Equal(t, original.Predefined, found.Predefined)
	assert.Equal(t, original.UserID, found.UserID)
	assert.WithinDuration(t, original.CreatedAt, found.CreatedAt, 1_000_000_000)
	assert.WithinDuration(t, original.UpdatedAt, found.UpdatedAt, 1_000_000_000)
}

// =============================================================================
// Archive / Unarchive
// =============================================================================

func TestPostgresPieceDefRepository_Archive_SetsArchivedAt(t *testing.T) {
	defRepo, userRepo := setupPieceDefRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	def := createTestPieceDef(t, defRepo, user.ID)
	assert.Nil(t, def.ArchivedAt)

	err := defRepo.Archive(ctx, def.ID)
	require.NoError(t, err)

	found, err := defRepo.GetByID(ctx, def.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.ArchivedAt, "ArchivedAt should be set after Archive")
}

func TestPostgresPieceDefRepository_Unarchive_ClearsArchivedAt(t *testing.T) {
	defRepo, userRepo := setupPieceDefRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	def := createTestPieceDef(t, defRepo, user.ID)

	// Archive first
	require.NoError(t, defRepo.Archive(ctx, def.ID))

	// Unarchive
	err := defRepo.Unarchive(ctx, def.ID)
	require.NoError(t, err)

	found, err := defRepo.GetByID(ctx, def.ID)
	require.NoError(t, err)
	assert.Nil(t, found.ArchivedAt, "ArchivedAt should be nil after Unarchive")
}

func TestPostgresPieceDefRepository_ListByUserID_ExcludesArchivedByDefault(t *testing.T) {
	defRepo, userRepo := setupPieceDefRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	activeDef := createTestPieceDef(t, defRepo, user.ID)
	archivedDef := createTestPieceDef(t, defRepo, user.ID)

	require.NoError(t, defRepo.Archive(ctx, archivedDef.ID))

	pg, _ := pagination.NewPagination(1, 20)
	defs, total, err := defRepo.ListByUserID(ctx, user.ID, false, pg)
	require.NoError(t, err)

	assert.Equal(t, int64(1), total)
	assert.Len(t, defs, 1)
	assert.Equal(t, activeDef.ID, defs[0].ID)
}

func TestPostgresPieceDefRepository_ListByUserID_IncludesArchivedWhenRequested(t *testing.T) {
	defRepo, userRepo := setupPieceDefRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	createTestPieceDef(t, defRepo, user.ID)
	archivedDef := createTestPieceDef(t, defRepo, user.ID)

	require.NoError(t, defRepo.Archive(ctx, archivedDef.ID))

	pg, _ := pagination.NewPagination(1, 20)
	defs, total, err := defRepo.ListByUserID(ctx, user.ID, true, pg)
	require.NoError(t, err)

	assert.Equal(t, int64(2), total)
	assert.Len(t, defs, 2)
}

func TestPostgresPieceDefRepository_Mapper_PreservesArchivedAt(t *testing.T) {
	defRepo, userRepo := setupPieceDefRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	def := createTestPieceDef(t, defRepo, user.ID)

	require.NoError(t, defRepo.Archive(ctx, def.ID))

	found, err := defRepo.GetByID(ctx, def.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.ArchivedAt)

	// Unarchive and verify round-trip
	require.NoError(t, defRepo.Unarchive(ctx, def.ID))

	found2, err := defRepo.GetByID(ctx, def.ID)
	require.NoError(t, err)
	assert.Nil(t, found2.ArchivedAt)
}
