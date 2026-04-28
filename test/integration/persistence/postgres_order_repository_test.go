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

// setupOrderRepo creates order, project, client, and user repos with a clean DB.
func setupOrderRepo(t *testing.T) (
	*persistence.PostgresOrderRepository,
	*persistence.PostgresProjectRepository,
	*persistence.PostgresClientRepository,
	*persistence.PostgresUserRepository,
) {
	db := helpers.SetupTestDB(t)
	helpers.CleanDB(t, db)
	return persistence.NewPostgresOrderRepository(db),
		persistence.NewPostgresProjectRepository(db),
		persistence.NewPostgresClientRepository(db),
		persistence.NewPostgresUserRepository(db)
}

// createTestProject is a helper that creates and persists a project for FK tests.
func createTestProjectForOrder(
	t *testing.T,
	projectRepo *persistence.PostgresProjectRepository,
	clientID uuid.UUID,
) *entities.Project {
	project, err := entities.NewProject(entities.CreateProjectParams{
		Name:     "Test Project " + uuid.New().String()[:8],
		ClientID: clientID,
	})
	require.NoError(t, err)
	require.NoError(t, projectRepo.Create(context.Background(), project))
	return project
}

// =============================================================================
// Create + GetByID
// =============================================================================

func TestPostgresOrderRepository_Create_And_GetByID(t *testing.T) {
	orderRepo, projectRepo, clientRepo, userRepo := setupOrderRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project := createTestProjectForOrder(t, projectRepo, client.ID)

	order, err := entities.NewOrder(entities.CreateOrderParams{
		Title:       "Steel beams – lot 3",
		Description: "First batch of structural steel",
		ProjectID:   project.ID,
	})
	require.NoError(t, err)

	err = orderRepo.Create(ctx, order)
	require.NoError(t, err)

	found, err := orderRepo.GetByID(ctx, order.ID)
	require.NoError(t, err)

	assert.Equal(t, order.ID, found.ID)
	assert.Equal(t, "Steel beams – lot 3", found.Title)
	assert.Equal(t, entities.OrderStatusPending, found.Status)
	assert.Equal(t, "First batch of structural steel", found.Description)
	assert.Equal(t, project.ID, found.ProjectID)
	assert.False(t, found.CreatedAt.IsZero())
	assert.False(t, found.UpdatedAt.IsZero())
}

func TestPostgresOrderRepository_Create_WithCompletedStatus(t *testing.T) {
	orderRepo, projectRepo, clientRepo, userRepo := setupOrderRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project := createTestProjectForOrder(t, projectRepo, client.ID)

	order, err := entities.NewOrder(entities.CreateOrderParams{
		Title:     "Completed order",
		Status:    "completed",
		ProjectID: project.ID,
	})
	require.NoError(t, err)

	require.NoError(t, orderRepo.Create(ctx, order))

	found, err := orderRepo.GetByID(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, entities.OrderStatusCompleted, found.Status)
}

// =============================================================================
// ListByProjectID
// =============================================================================

func TestPostgresOrderRepository_ListByProjectID(t *testing.T) {
	orderRepo, projectRepo, clientRepo, userRepo := setupOrderRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project := createTestProjectForOrder(t, projectRepo, client.ID)

	o1, _ := entities.NewOrder(entities.CreateOrderParams{Title: "Order A", ProjectID: project.ID})
	o2, _ := entities.NewOrder(entities.CreateOrderParams{Title: "Order B", ProjectID: project.ID})
	require.NoError(t, orderRepo.Create(ctx, o1))
	require.NoError(t, orderRepo.Create(ctx, o2))

	pg, _ := pagination.NewPagination(1, 20)
	orders, total, err := orderRepo.ListByProjectID(ctx, project.ID, pg)
	require.NoError(t, err)
	assert.Len(t, orders, 2)
	assert.Equal(t, int64(2), total)
}

func TestPostgresOrderRepository_ListByProjectID_Empty(t *testing.T) {
	orderRepo, projectRepo, clientRepo, userRepo := setupOrderRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project := createTestProjectForOrder(t, projectRepo, client.ID)

	pg, _ := pagination.NewPagination(1, 20)
	orders, total, err := orderRepo.ListByProjectID(ctx, project.ID, pg)
	require.NoError(t, err)
	assert.Empty(t, orders)
	assert.Equal(t, int64(0), total)
}

func TestPostgresOrderRepository_ListByProjectID_DoesNotReturnOtherProjectsOrders(t *testing.T) {
	orderRepo, projectRepo, clientRepo, userRepo := setupOrderRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project1 := createTestProjectForOrder(t, projectRepo, client.ID)
	project2 := createTestProjectForOrder(t, projectRepo, client.ID)

	oA, _ := entities.NewOrder(entities.CreateOrderParams{Title: "Order A", ProjectID: project1.ID})
	oB, _ := entities.NewOrder(entities.CreateOrderParams{Title: "Order B", ProjectID: project2.ID})
	require.NoError(t, orderRepo.Create(ctx, oA))
	require.NoError(t, orderRepo.Create(ctx, oB))

	pg, _ := pagination.NewPagination(1, 20)

	orders1, _, err := orderRepo.ListByProjectID(ctx, project1.ID, pg)
	require.NoError(t, err)
	assert.Len(t, orders1, 1)
	assert.Equal(t, project1.ID, orders1[0].ProjectID)

	orders2, _, err := orderRepo.ListByProjectID(ctx, project2.ID, pg)
	require.NoError(t, err)
	assert.Len(t, orders2, 1)
	assert.Equal(t, project2.ID, orders2[0].ProjectID)
}

// =============================================================================
// Update
// =============================================================================

func TestPostgresOrderRepository_Update(t *testing.T) {
	orderRepo, projectRepo, clientRepo, userRepo := setupOrderRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project := createTestProjectForOrder(t, projectRepo, client.ID)
	order, _ := entities.NewOrder(
		entities.CreateOrderParams{Title: "Old Title", Description: "Old desc", ProjectID: project.ID},
	)
	require.NoError(t, orderRepo.Create(ctx, order))

	require.NoError(t, order.SetTitle("New Title"))
	require.NoError(t, order.SetStatus("completed"))
	order.SetDescription("New desc")
	err := orderRepo.Update(ctx, order)
	require.NoError(t, err)

	found, err := orderRepo.GetByID(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Title", found.Title)
	assert.Equal(t, entities.OrderStatusCompleted, found.Status)
	assert.Equal(t, "New desc", found.Description)
}

// =============================================================================
// Delete
// =============================================================================

func TestPostgresOrderRepository_Delete(t *testing.T) {
	orderRepo, projectRepo, clientRepo, userRepo := setupOrderRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project := createTestProjectForOrder(t, projectRepo, client.ID)
	order, _ := entities.NewOrder(entities.CreateOrderParams{Title: "To Delete", ProjectID: project.ID})
	require.NoError(t, orderRepo.Create(ctx, order))

	err := orderRepo.Delete(ctx, order.ID)
	require.NoError(t, err)

	found, err := orderRepo.GetByID(ctx, order.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

// =============================================================================
// GetByID — Not Found
// =============================================================================

func TestPostgresOrderRepository_GetByID_NotFound(t *testing.T) {
	orderRepo, _, _, _ := setupOrderRepo(t)
	ctx := context.Background()

	found, err := orderRepo.GetByID(ctx, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, found)
}

// =============================================================================
// Create — FK violation (non-existing project)
// =============================================================================

func TestPostgresOrderRepository_Create_WithInvalidProjectID_Fails(t *testing.T) {
	orderRepo, _, _, _ := setupOrderRepo(t)
	ctx := context.Background()

	order, _ := entities.NewOrder(entities.CreateOrderParams{Title: "Orphan Order", ProjectID: uuid.New()})
	err := orderRepo.Create(ctx, order)

	assert.Error(t, err, "creating an order with a non-existing project_id should fail due to FK constraint")
}

// =============================================================================
// Two Projects Same Order Title — Both succeed
// =============================================================================

func TestPostgresOrderRepository_TwoProjects_SameOrderTitle_BothSucceed(t *testing.T) {
	orderRepo, projectRepo, clientRepo, userRepo := setupOrderRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project1 := createTestProjectForOrder(t, projectRepo, client.ID)
	project2 := createTestProjectForOrder(t, projectRepo, client.ID)

	oA, _ := entities.NewOrder(entities.CreateOrderParams{Title: "Same Title", ProjectID: project1.ID})
	oB, _ := entities.NewOrder(entities.CreateOrderParams{Title: "Same Title", ProjectID: project2.ID})

	require.NoError(t, orderRepo.Create(ctx, oA))
	require.NoError(t, orderRepo.Create(ctx, oB))

	foundA, err := orderRepo.GetByID(ctx, oA.ID)
	require.NoError(t, err)
	assert.Equal(t, "Same Title", foundA.Title)
	assert.Equal(t, project1.ID, foundA.ProjectID)

	foundB, err := orderRepo.GetByID(ctx, oB.ID)
	require.NoError(t, err)
	assert.Equal(t, "Same Title", foundB.Title)
	assert.Equal(t, project2.ID, foundB.ProjectID)

	assert.NotEqual(t, foundA.ID, foundB.ID)
}

// =============================================================================
// Mapper — Data integrity
// =============================================================================

func TestPostgresOrderRepository_Mapper_PreservesAllFields(t *testing.T) {
	orderRepo, projectRepo, clientRepo, userRepo := setupOrderRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project := createTestProjectForOrder(t, projectRepo, client.ID)
	original, _ := entities.NewOrder(
		entities.CreateOrderParams{Title: "Full Data Order", Description: "Full description", ProjectID: project.ID},
	)
	require.NoError(t, orderRepo.Create(ctx, original))

	found, err := orderRepo.GetByID(ctx, original.ID)
	require.NoError(t, err)

	assert.Equal(t, original.ID, found.ID)
	assert.Equal(t, original.Title, found.Title)
	assert.Equal(t, original.Status, found.Status)
	assert.Equal(t, original.Description, found.Description)
	assert.Equal(t, original.ProjectID, found.ProjectID)
	assert.WithinDuration(t, original.CreatedAt, found.CreatedAt, 1_000_000_000)
	assert.WithinDuration(t, original.UpdatedAt, found.UpdatedAt, 1_000_000_000)
}

// =============================================================================
// Cascade Delete — Deleting project deletes their orders
// =============================================================================

func TestPostgresOrderRepository_CascadeDelete_ProjectDeletion(t *testing.T) {
	orderRepo, projectRepo, clientRepo, userRepo := setupOrderRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project := createTestProjectForOrder(t, projectRepo, client.ID)
	order, _ := entities.NewOrder(entities.CreateOrderParams{Title: "Will Be Orphaned", ProjectID: project.ID})
	require.NoError(t, orderRepo.Create(ctx, order))

	// Delete the project directly via DB (simulating cascade)
	db := helpers.SetupTestDB(t)
	err := db.Exec("DELETE FROM projects WHERE id = ?", project.ID).Error
	require.NoError(t, err)

	// The order should be gone too (CASCADE)
	found, err := orderRepo.GetByID(ctx, order.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}
