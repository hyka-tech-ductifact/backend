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

// setupPieceRepo creates all repos needed for piece persistence tests.
func setupPieceRepo(t *testing.T) (
	*persistence.PostgresPieceRepository,
	*persistence.PostgresPieceDefinitionRepository,
	*persistence.PostgresOrderRepository,
	*persistence.PostgresProjectRepository,
	*persistence.PostgresClientRepository,
	*persistence.PostgresUserRepository,
) {
	db := helpers.SetupTestDB(t)
	helpers.CleanDB(t, db)
	return persistence.NewPostgresPieceRepository(db),
		persistence.NewPostgresPieceDefinitionRepository(db),
		persistence.NewPostgresOrderRepository(db),
		persistence.NewPostgresProjectRepository(db),
		persistence.NewPostgresClientRepository(db),
		persistence.NewPostgresUserRepository(db)
}

// createTestOrderForPiece sets up the full FK chain: user → client → project → order.
func createTestOrderForPiece(
	t *testing.T,
	userRepo *persistence.PostgresUserRepository,
	clientRepo *persistence.PostgresClientRepository,
	projectRepo *persistence.PostgresProjectRepository,
	orderRepo *persistence.PostgresOrderRepository,
) (*entities.User, *entities.Order) {
	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project := createTestProjectForOrder(t, projectRepo, client.ID)
	order, err := entities.NewOrder(entities.CreateOrderParams{
		Title:     "Test Order " + uuid.New().String()[:8],
		ProjectID: project.ID,
	})
	require.NoError(t, err)
	require.NoError(t, orderRepo.Create(context.Background(), order))
	return user, order
}

// =============================================================================
// Create + GetByID
// =============================================================================

func TestPostgresPieceRepository_Create_And_GetByID(t *testing.T) {
	pieceRepo, defRepo, orderRepo, projectRepo, clientRepo, userRepo := setupPieceRepo(t)
	ctx := context.Background()

	user, order := createTestOrderForPiece(t, userRepo, clientRepo, projectRepo, orderRepo)
	def := createTestPieceDef(t, defRepo, user.ID)

	piece, err := entities.NewPiece(entities.CreatePieceParams{
		Title:        "Side panel",
		OrderID:      order.ID,
		DefinitionID: def.ID,
		Dimensions:   map[string]float64{"Length": 150.5, "Width": 80.0},
		Quantity:     15,
	})
	require.NoError(t, err)

	err = pieceRepo.Create(ctx, piece)
	require.NoError(t, err)

	found, err := pieceRepo.GetByID(ctx, piece.ID)
	require.NoError(t, err)

	assert.Equal(t, piece.ID, found.ID)
	assert.Equal(t, "Side panel", found.Title)
	assert.Equal(t, order.ID, found.OrderID)
	assert.Equal(t, def.ID, found.DefinitionID)
	assert.Equal(t, 150.5, found.Dimensions["Length"])
	assert.Equal(t, 80.0, found.Dimensions["Width"])
	assert.Equal(t, 15, found.Quantity)
	assert.False(t, found.CreatedAt.IsZero())
	assert.False(t, found.UpdatedAt.IsZero())
}

// =============================================================================
// ListByOrderID
// =============================================================================

func TestPostgresPieceRepository_ListByOrderID(t *testing.T) {
	pieceRepo, defRepo, orderRepo, projectRepo, clientRepo, userRepo := setupPieceRepo(t)
	ctx := context.Background()

	user, order := createTestOrderForPiece(t, userRepo, clientRepo, projectRepo, orderRepo)
	def := createTestPieceDef(t, defRepo, user.ID)

	p1, _ := entities.NewPiece(entities.CreatePieceParams{
		Title: "Panel A", OrderID: order.ID, DefinitionID: def.ID,
		Dimensions: map[string]float64{"Length": 100, "Width": 50}, Quantity: 1,
	})
	p2, _ := entities.NewPiece(entities.CreatePieceParams{
		Title: "Panel B", OrderID: order.ID, DefinitionID: def.ID,
		Dimensions: map[string]float64{"Length": 200, "Width": 100}, Quantity: 2,
	})
	require.NoError(t, pieceRepo.Create(ctx, p1))
	require.NoError(t, pieceRepo.Create(ctx, p2))

	pg, _ := pagination.NewPagination(1, 20)
	pieces, total, err := pieceRepo.ListByOrderID(ctx, order.ID, pg)
	require.NoError(t, err)
	assert.Len(t, pieces, 2)
	assert.Equal(t, int64(2), total)
}

func TestPostgresPieceRepository_ListByOrderID_Empty(t *testing.T) {
	pieceRepo, _, orderRepo, projectRepo, clientRepo, userRepo := setupPieceRepo(t)
	ctx := context.Background()

	_, order := createTestOrderForPiece(t, userRepo, clientRepo, projectRepo, orderRepo)

	pg, _ := pagination.NewPagination(1, 20)
	pieces, total, err := pieceRepo.ListByOrderID(ctx, order.ID, pg)
	require.NoError(t, err)
	assert.Empty(t, pieces)
	assert.Equal(t, int64(0), total)
}

func TestPostgresPieceRepository_ListByOrderID_DoesNotReturnOtherOrdersPieces(t *testing.T) {
	pieceRepo, defRepo, orderRepo, projectRepo, clientRepo, userRepo := setupPieceRepo(t)
	ctx := context.Background()

	user, order1 := createTestOrderForPiece(t, userRepo, clientRepo, projectRepo, orderRepo)
	_, order2 := createTestOrderForPiece(t, userRepo, clientRepo, projectRepo, orderRepo)
	def := createTestPieceDef(t, defRepo, user.ID)

	pA, _ := entities.NewPiece(entities.CreatePieceParams{
		Title: "Panel A", OrderID: order1.ID, DefinitionID: def.ID,
		Dimensions: map[string]float64{"Length": 100, "Width": 50}, Quantity: 1,
	})
	pB, _ := entities.NewPiece(entities.CreatePieceParams{
		Title: "Panel B", OrderID: order2.ID, DefinitionID: def.ID,
		Dimensions: map[string]float64{"Length": 200, "Width": 100}, Quantity: 2,
	})
	require.NoError(t, pieceRepo.Create(ctx, pA))
	require.NoError(t, pieceRepo.Create(ctx, pB))

	pg, _ := pagination.NewPagination(1, 20)

	pieces1, _, err := pieceRepo.ListByOrderID(ctx, order1.ID, pg)
	require.NoError(t, err)
	assert.Len(t, pieces1, 1)
	assert.Equal(t, order1.ID, pieces1[0].OrderID)

	pieces2, _, err := pieceRepo.ListByOrderID(ctx, order2.ID, pg)
	require.NoError(t, err)
	assert.Len(t, pieces2, 1)
	assert.Equal(t, order2.ID, pieces2[0].OrderID)
}

// =============================================================================
// Update
// =============================================================================

func TestPostgresPieceRepository_Update(t *testing.T) {
	pieceRepo, defRepo, orderRepo, projectRepo, clientRepo, userRepo := setupPieceRepo(t)
	ctx := context.Background()

	user, order := createTestOrderForPiece(t, userRepo, clientRepo, projectRepo, orderRepo)
	def := createTestPieceDef(t, defRepo, user.ID)
	piece, _ := entities.NewPiece(entities.CreatePieceParams{
		Title: "Old Title", OrderID: order.ID, DefinitionID: def.ID,
		Dimensions: map[string]float64{"Length": 100, "Width": 50}, Quantity: 1,
	})
	require.NoError(t, pieceRepo.Create(ctx, piece))

	require.NoError(t, piece.SetTitle("New Title"))
	require.NoError(t, piece.SetQuantity(25))
	piece.Dimensions = map[string]float64{"Length": 200, "Width": 100}

	err := pieceRepo.Update(ctx, piece)
	require.NoError(t, err)

	found, err := pieceRepo.GetByID(ctx, piece.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Title", found.Title)
	assert.Equal(t, 25, found.Quantity)
	assert.Equal(t, 200.0, found.Dimensions["Length"])
	assert.Equal(t, 100.0, found.Dimensions["Width"])
}

// =============================================================================
// Delete
// =============================================================================

func TestPostgresPieceRepository_Delete(t *testing.T) {
	pieceRepo, defRepo, orderRepo, projectRepo, clientRepo, userRepo := setupPieceRepo(t)
	ctx := context.Background()

	user, order := createTestOrderForPiece(t, userRepo, clientRepo, projectRepo, orderRepo)
	def := createTestPieceDef(t, defRepo, user.ID)
	piece, _ := entities.NewPiece(entities.CreatePieceParams{
		Title: "To Delete", OrderID: order.ID, DefinitionID: def.ID,
		Dimensions: map[string]float64{"Length": 100, "Width": 50}, Quantity: 1,
	})
	require.NoError(t, pieceRepo.Create(ctx, piece))

	err := pieceRepo.Delete(ctx, piece.ID)
	require.NoError(t, err)

	found, err := pieceRepo.GetByID(ctx, piece.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

// =============================================================================
// GetByID — Not Found
// =============================================================================

func TestPostgresPieceRepository_GetByID_NotFound(t *testing.T) {
	pieceRepo, _, _, _, _, _ := setupPieceRepo(t)
	ctx := context.Background()

	found, err := pieceRepo.GetByID(ctx, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, found)
}

// =============================================================================
// Create — FK violation (non-existing order)
// =============================================================================

func TestPostgresPieceRepository_Create_WithInvalidOrderID_Fails(t *testing.T) {
	pieceRepo, defRepo, _, _, _, userRepo := setupPieceRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	def := createTestPieceDef(t, defRepo, user.ID)

	piece, _ := entities.NewPiece(entities.CreatePieceParams{
		Title: "Orphan Piece", OrderID: uuid.New(), DefinitionID: def.ID,
		Dimensions: map[string]float64{"Length": 100, "Width": 50}, Quantity: 1,
	})
	err := pieceRepo.Create(ctx, piece)

	assert.Error(t, err, "creating a piece with a non-existing order_id should fail due to FK constraint")
}

// =============================================================================
// Cascade Delete — Deleting order deletes its pieces
// =============================================================================

func TestPostgresPieceRepository_CascadeDelete_OrderDeletion(t *testing.T) {
	pieceRepo, defRepo, orderRepo, projectRepo, clientRepo, userRepo := setupPieceRepo(t)
	ctx := context.Background()

	user, order := createTestOrderForPiece(t, userRepo, clientRepo, projectRepo, orderRepo)
	def := createTestPieceDef(t, defRepo, user.ID)
	piece, _ := entities.NewPiece(entities.CreatePieceParams{
		Title: "Will Be Orphaned", OrderID: order.ID, DefinitionID: def.ID,
		Dimensions: map[string]float64{"Length": 100, "Width": 50}, Quantity: 1,
	})
	require.NoError(t, pieceRepo.Create(ctx, piece))

	// Delete the order directly via DB (simulating cascade)
	db := helpers.SetupTestDB(t)
	err := db.Exec("DELETE FROM orders WHERE id = ?", order.ID).Error
	require.NoError(t, err)

	// The piece should be gone too (CASCADE)
	found, err := pieceRepo.GetByID(ctx, piece.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

// =============================================================================
// Mapper — Data integrity
// =============================================================================

func TestPostgresPieceRepository_Mapper_PreservesAllFields(t *testing.T) {
	pieceRepo, defRepo, orderRepo, projectRepo, clientRepo, userRepo := setupPieceRepo(t)
	ctx := context.Background()

	user, order := createTestOrderForPiece(t, userRepo, clientRepo, projectRepo, orderRepo)
	def := createTestPieceDef(t, defRepo, user.ID)
	original, _ := entities.NewPiece(entities.CreatePieceParams{
		Title: "Full Data Piece", OrderID: order.ID, DefinitionID: def.ID,
		Dimensions: map[string]float64{"Length": 123.456, "Width": 78.9}, Quantity: 42,
	})
	require.NoError(t, pieceRepo.Create(ctx, original))

	found, err := pieceRepo.GetByID(ctx, original.ID)
	require.NoError(t, err)

	assert.Equal(t, original.ID, found.ID)
	assert.Equal(t, original.Title, found.Title)
	assert.Equal(t, original.OrderID, found.OrderID)
	assert.Equal(t, original.DefinitionID, found.DefinitionID)
	assert.Equal(t, original.Dimensions, found.Dimensions)
	assert.Equal(t, original.Quantity, found.Quantity)
	assert.WithinDuration(t, original.CreatedAt, found.CreatedAt, 1_000_000_000)
	assert.WithinDuration(t, original.UpdatedAt, found.UpdatedAt, 1_000_000_000)
}
