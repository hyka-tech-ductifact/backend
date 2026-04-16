package services_test

import (
"context"
"errors"
"testing"

"ductifact/internal/application/services"
"ductifact/internal/domain/entities"
"ductifact/internal/domain/pagination"
"ductifact/internal/domain/repositories"
"ductifact/test/unit/mocks"

"github.com/google/uuid"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

// =============================================================================
// CreatePiece
// =============================================================================

func TestCreatePiece_WithValidData_ReturnsPiece(t *testing.T) {
userID := uuid.New()
order := newTestOrder(uuid.New())
def := newTestPieceDef(userID)

svc := services.NewPieceService(
&mocks.MockPieceRepository{},
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

piece, err := svc.CreatePiece(context.Background(), userID, entities.CreatePieceParams{
Title:        "Side panel",
OrderID:      order.ID,
DefinitionID: def.ID,
Dimensions:   map[string]float64{"Length": 150.5, "Width": 80.0},
Quantity:     15,
})

require.NoError(t, err)
assert.Equal(t, "Side panel", piece.Title)
assert.Equal(t, 15, piece.Quantity)
assert.Equal(t, order.ID, piece.OrderID)
assert.Equal(t, def.ID, piece.DefinitionID)
}

func TestCreatePiece_WithEmptyTitle_ReturnsError(t *testing.T) {
userID := uuid.New()
order := newTestOrder(uuid.New())
def := newTestPieceDef(userID)

svc := services.NewPieceService(
&mocks.MockPieceRepository{},
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

piece, err := svc.CreatePiece(context.Background(), userID, entities.CreatePieceParams{
OrderID:      order.ID,
DefinitionID: def.ID,
Dimensions:   map[string]float64{"Length": 150.5, "Width": 80.0},
Quantity:     15,
})

assert.Nil(t, piece)
assert.ErrorIs(t, err, entities.ErrEmptyPieceTitle)
}

func TestCreatePiece_WithUnexpectedDimensions_ReturnsError(t *testing.T) {
userID := uuid.New()
order := newTestOrder(uuid.New())
def := newTestPieceDef(userID) // schema: ["Length", "Width"]

svc := services.NewPieceService(
&mocks.MockPieceRepository{},
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

piece, err := svc.CreatePiece(context.Background(), userID, entities.CreatePieceParams{
Title:        "Panel",
OrderID:      order.ID,
DefinitionID: def.ID,
Dimensions:   map[string]float64{"Length": 100, "Width": 50, "Height": 30},
Quantity:     1,
})

assert.Nil(t, piece)
assert.ErrorIs(t, err, entities.ErrUnexpectedDimensions)
}

func TestCreatePiece_WithMissingDimensions_ReturnsError(t *testing.T) {
userID := uuid.New()
order := newTestOrder(uuid.New())
def := newTestPieceDef(userID) // schema: ["Length", "Width"]

svc := services.NewPieceService(
&mocks.MockPieceRepository{},
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

piece, err := svc.CreatePiece(context.Background(), userID, entities.CreatePieceParams{
Title:        "Panel",
OrderID:      order.ID,
DefinitionID: def.ID,
Dimensions:   map[string]float64{"Length": 100},
Quantity:     1,
})

assert.Nil(t, piece)
assert.ErrorIs(t, err, entities.ErrMissingDimensions)
}

func TestCreatePiece_WithNotOwnedOrder_ReturnsError(t *testing.T) {
orderRepo := &mocks.MockOrderRepository{
GetByIDForOwnerFn: func(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Order, error) {
return nil, repositories.ErrOrderNotOwned
},
}

svc := services.NewPieceService(
&mocks.MockPieceRepository{},
&mocks.MockPieceDefinitionRepository{},
orderRepo,
)

piece, err := svc.CreatePiece(context.Background(), uuid.New(), entities.CreatePieceParams{
Title: "Panel", OrderID: uuid.New(), DefinitionID: uuid.New(),
Dimensions: map[string]float64{"L": 1}, Quantity: 1,
})

assert.Nil(t, piece)
assert.ErrorIs(t, err, repositories.ErrOrderNotOwned)
}

func TestCreatePiece_WithNonExistingPieceDef_ReturnsError(t *testing.T) {
userID := uuid.New()
order := newTestOrder(uuid.New())

svc := services.NewPieceService(
&mocks.MockPieceRepository{},
pieceDefRepoReturning(nil),
orderRepoReturning(order),
)

piece, err := svc.CreatePiece(context.Background(), userID, entities.CreatePieceParams{
Title: "Panel", OrderID: order.ID, DefinitionID: uuid.New(),
Dimensions: map[string]float64{"L": 1}, Quantity: 1,
})

assert.Nil(t, piece)
assert.ErrorIs(t, err, services.ErrPieceDefNotFound)
}

func TestCreatePiece_WhenRepoFails_ReturnsError(t *testing.T) {
userID := uuid.New()
order := newTestOrder(uuid.New())
def := newTestPieceDef(userID)

pieceRepo := &mocks.MockPieceRepository{
CreateFn: func(ctx context.Context, piece *entities.Piece) error {
return errors.New("db connection lost")
},
}
svc := services.NewPieceService(
pieceRepo,
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

piece, err := svc.CreatePiece(context.Background(), userID, entities.CreatePieceParams{
Title: "Panel", OrderID: order.ID, DefinitionID: def.ID,
Dimensions: map[string]float64{"Length": 100, "Width": 50}, Quantity: 1,
})

assert.Nil(t, piece)
assert.EqualError(t, err, "db connection lost")
}

// =============================================================================
// GetPieceByID
// =============================================================================

func TestGetPieceByID_WithExistingPiece_ReturnsPiece(t *testing.T) {
userID := uuid.New()
def := newTestPieceDef(userID)
order := newTestOrder(uuid.New())
piece := newTestPiece(order.ID, def.ID)

svc := services.NewPieceService(
pieceRepoReturning(piece),
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

result, err := svc.GetPieceByID(context.Background(), piece.ID, userID)

require.NoError(t, err)
assert.Equal(t, piece.Title, result.Title)
assert.Equal(t, piece.ID, result.ID)
}

func TestGetPieceByID_NotFound_ReturnsError(t *testing.T) {
userID := uuid.New()

svc := services.NewPieceService(
pieceRepoReturning(nil),
&mocks.MockPieceDefinitionRepository{},
&mocks.MockOrderRepository{},
)

result, err := svc.GetPieceByID(context.Background(), uuid.New(), userID)

assert.Nil(t, result)
assert.ErrorIs(t, err, services.ErrPieceNotFound)
}

func TestGetPieceByID_WithNotOwned_ReturnsError(t *testing.T) {
pieceRepo := &mocks.MockPieceRepository{
GetByIDForOwnerFn: func(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Piece, error) {
return nil, repositories.ErrPieceNotOwned
},
}

svc := services.NewPieceService(
pieceRepo,
&mocks.MockPieceDefinitionRepository{},
&mocks.MockOrderRepository{},
)

result, err := svc.GetPieceByID(context.Background(), uuid.New(), uuid.New())

assert.Nil(t, result)
assert.ErrorIs(t, err, repositories.ErrPieceNotOwned)
}

// =============================================================================
// ListPiecesByOrderID
// =============================================================================

func TestListPiecesByOrderID_ReturnsPieces(t *testing.T) {
userID := uuid.New()
order := newTestOrder(uuid.New())
expected := []*entities.Piece{
{ID: uuid.New(), Title: "Panel A", OrderID: order.ID},
{ID: uuid.New(), Title: "Panel B", OrderID: order.ID},
}
pieceRepo := &mocks.MockPieceRepository{
ListByOrderIDFn: func(ctx context.Context, oID uuid.UUID, pg pagination.Pagination) ([]*entities.Piece, int64, error) {
return expected, 2, nil
},
}
svc := services.NewPieceService(
pieceRepo,
&mocks.MockPieceDefinitionRepository{},
orderRepoReturning(order),
)

pg, _ := pagination.NewPagination(1, 20)
result, err := svc.ListPiecesByOrderID(context.Background(), order.ID, userID, pg)

require.NoError(t, err)
assert.Len(t, result.Data, 2)
assert.Equal(t, int64(2), result.TotalItems)
}

func TestListPiecesByOrderID_EmptyList(t *testing.T) {
userID := uuid.New()
order := newTestOrder(uuid.New())
pieceRepo := &mocks.MockPieceRepository{
ListByOrderIDFn: func(ctx context.Context, oID uuid.UUID, pg pagination.Pagination) ([]*entities.Piece, int64, error) {
return []*entities.Piece{}, 0, nil
},
}
svc := services.NewPieceService(
pieceRepo,
&mocks.MockPieceDefinitionRepository{},
orderRepoReturning(order),
)

pg, _ := pagination.NewPagination(1, 20)
result, err := svc.ListPiecesByOrderID(context.Background(), order.ID, userID, pg)

require.NoError(t, err)
assert.Empty(t, result.Data)
assert.Equal(t, int64(0), result.TotalItems)
}

func TestListPiecesByOrderID_WithWrongUser_ReturnsError(t *testing.T) {
orderRepo := &mocks.MockOrderRepository{
GetByIDForOwnerFn: func(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Order, error) {
return nil, repositories.ErrOrderNotOwned
},
}

svc := services.NewPieceService(
&mocks.MockPieceRepository{},
&mocks.MockPieceDefinitionRepository{},
orderRepo,
)

pg, _ := pagination.NewPagination(1, 20)
_, err := svc.ListPiecesByOrderID(context.Background(), uuid.New(), uuid.New(), pg)

assert.ErrorIs(t, err, repositories.ErrOrderNotOwned)
}

// =============================================================================
// UpdatePiece
// =============================================================================

func TestUpdatePiece_WithNewTitle_Updates(t *testing.T) {
userID := uuid.New()
def := newTestPieceDef(userID)
order := newTestOrder(uuid.New())
piece := newTestPiece(order.ID, def.ID)

svc := services.NewPieceService(
pieceRepoReturning(piece),
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

result, err := svc.UpdatePiece(context.Background(), piece.ID, userID, entities.UpdatePieceParams{
Title: strPtr("Updated panel"),
})

require.NoError(t, err)
assert.Equal(t, "Updated panel", result.Title)
}

func TestUpdatePiece_WithNewQuantity_Updates(t *testing.T) {
userID := uuid.New()
def := newTestPieceDef(userID)
order := newTestOrder(uuid.New())
piece := newTestPiece(order.ID, def.ID)

svc := services.NewPieceService(
pieceRepoReturning(piece),
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

result, err := svc.UpdatePiece(context.Background(), piece.ID, userID, entities.UpdatePieceParams{
Quantity: intPtr(25),
})

require.NoError(t, err)
assert.Equal(t, 25, result.Quantity)
}

func TestUpdatePiece_WithNewDimensions_RevalidatesSchema(t *testing.T) {
userID := uuid.New()
def := newTestPieceDef(userID) // schema: ["Length", "Width"]
order := newTestOrder(uuid.New())
piece := newTestPiece(order.ID, def.ID)

svc := services.NewPieceService(
pieceRepoReturning(piece),
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

result, err := svc.UpdatePiece(context.Background(), piece.ID, userID, entities.UpdatePieceParams{
Dimensions: dimsPtr(map[string]float64{"Length": 200, "Width": 100}),
})

require.NoError(t, err)
assert.Equal(t, 200.0, result.Dimensions["Length"])
assert.Equal(t, 100.0, result.Dimensions["Width"])
}

func TestUpdatePiece_WithInvalidDimensions_ReturnsError(t *testing.T) {
userID := uuid.New()
def := newTestPieceDef(userID) // schema: ["Length", "Width"]
order := newTestOrder(uuid.New())
piece := newTestPiece(order.ID, def.ID)

svc := services.NewPieceService(
pieceRepoReturning(piece),
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

result, err := svc.UpdatePiece(context.Background(), piece.ID, userID, entities.UpdatePieceParams{
Dimensions: dimsPtr(map[string]float64{"Length": 200, "Width": 100, "Height": 50}),
})

assert.Nil(t, result)
assert.ErrorIs(t, err, entities.ErrUnexpectedDimensions)
}

func TestUpdatePiece_WithEmptyTitle_ReturnsError(t *testing.T) {
userID := uuid.New()
def := newTestPieceDef(userID)
order := newTestOrder(uuid.New())
piece := newTestPiece(order.ID, def.ID)

svc := services.NewPieceService(
pieceRepoReturning(piece),
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

result, err := svc.UpdatePiece(context.Background(), piece.ID, userID, entities.UpdatePieceParams{
Title: strPtr(""),
})

assert.Nil(t, result)
assert.ErrorIs(t, err, entities.ErrEmptyPieceTitle)
}

func TestUpdatePiece_WithNoChanges_SkipsPersistence(t *testing.T) {
userID := uuid.New()
def := newTestPieceDef(userID)
order := newTestOrder(uuid.New())
piece := newTestPiece(order.ID, def.ID)
repo := pieceRepoReturning(piece)
updateCalled := false
repo.UpdateFn = func(ctx context.Context, p *entities.Piece) error {
updateCalled = true
return nil
}
svc := services.NewPieceService(
repo,
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

_, err := svc.UpdatePiece(context.Background(), piece.ID, userID, entities.UpdatePieceParams{})

require.NoError(t, err)
assert.False(t, updateCalled, "Update should not be called when no fields change")
}

func TestUpdatePiece_UpdatesTimestamp(t *testing.T) {
userID := uuid.New()
def := newTestPieceDef(userID)
order := newTestOrder(uuid.New())
piece := newTestPiece(order.ID, def.ID)
oldTime := piece.UpdatedAt

svc := services.NewPieceService(
pieceRepoReturning(piece),
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

result, err := svc.UpdatePiece(context.Background(), piece.ID, userID, entities.UpdatePieceParams{
Title: strPtr("New Title"),
})

require.NoError(t, err)
assert.True(t, result.UpdatedAt.After(oldTime), "UpdatedAt must be newer than the original")
}

// =============================================================================
// DeletePiece
// =============================================================================

func TestDeletePiece_WithExistingPiece_Succeeds(t *testing.T) {
userID := uuid.New()
def := newTestPieceDef(userID)
order := newTestOrder(uuid.New())
piece := newTestPiece(order.ID, def.ID)
deleteCalled := false
repo := pieceRepoReturning(piece)
repo.DeleteFn = func(ctx context.Context, id uuid.UUID) error {
deleteCalled = true
return nil
}
svc := services.NewPieceService(
repo,
pieceDefRepoReturning(def),
orderRepoReturning(order),
)

err := svc.DeletePiece(context.Background(), piece.ID, userID)

assert.NoError(t, err)
assert.True(t, deleteCalled)
}

func TestDeletePiece_NotFound_ReturnsError(t *testing.T) {
userID := uuid.New()

svc := services.NewPieceService(
pieceRepoReturning(nil),
&mocks.MockPieceDefinitionRepository{},
&mocks.MockOrderRepository{},
)

err := svc.DeletePiece(context.Background(), uuid.New(), userID)

assert.ErrorIs(t, err, services.ErrPieceNotFound)
}

func TestDeletePiece_WithWrongUser_ReturnsError(t *testing.T) {
pieceRepo := &mocks.MockPieceRepository{
GetByIDForOwnerFn: func(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Piece, error) {
return nil, repositories.ErrPieceNotOwned
},
}

svc := services.NewPieceService(
pieceRepo,
&mocks.MockPieceDefinitionRepository{},
&mocks.MockOrderRepository{},
)

err := svc.DeletePiece(context.Background(), uuid.New(), uuid.New())

assert.ErrorIs(t, err, repositories.ErrPieceNotOwned)
}
