package persistence_test

import (
	"context"
	"testing"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/query"
	"ductifact/internal/infrastructure/adapters/outbound/persistence"
	"ductifact/test/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func projectListQuery(pg query.PageRequest) query.ProjectListQuery {
	return query.ProjectListQuery{ListQuery: query.ListQuery{Page: pg}}
}

// setupProjectRepo creates client, user, and project repos with a clean DB.
func setupProjectRepo(
	t *testing.T,
) (*persistence.PostgresProjectRepository, *persistence.PostgresClientRepository, *persistence.PostgresUserRepository) {
	db := helpers.SetupTestDB(t)
	helpers.CleanDB(t, db)
	return persistence.NewPostgresProjectRepository(db),
		persistence.NewPostgresClientRepository(db),
		persistence.NewPostgresUserRepository(db)
}

// createTestClient is a helper that creates and persists a client for FK tests.
func createTestClient(
	t *testing.T,
	clientRepo *persistence.PostgresClientRepository,
	userID uuid.UUID,
) *entities.Client {
	client, err := entities.NewClient(entities.CreateClientParams{
		Name:   "Test Client " + uuid.New().String()[:8],
		UserID: userID,
	})
	require.NoError(t, err)
	require.NoError(t, clientRepo.Create(context.Background(), client))
	return client
}

// newProjectParams is a helper that returns CreateProjectParams with all fields populated.
func newProjectParams(name string, clientID uuid.UUID) entities.CreateProjectParams {
	return entities.CreateProjectParams{
		Name:        name,
		Address:     "Calle Mayor 12, Madrid",
		ManagerName: "Carlos Pérez",
		Phone:       "+34 699 111 222",
		Description: "Test project description",
		ClientID:    clientID,
	}
}

// =============================================================================
// Create + GetByID
// =============================================================================

func TestPostgresProjectRepository_Create_And_GetByID(t *testing.T) {
	projectRepo, clientRepo, userRepo := setupProjectRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)

	project, err := entities.NewProject(newProjectParams("Residential Tower B", client.ID))
	require.NoError(t, err)

	err = projectRepo.Create(ctx, project)
	require.NoError(t, err)

	found, err := projectRepo.GetByID(ctx, project.ID)
	require.NoError(t, err)

	assert.Equal(t, project.ID, found.ID)
	assert.Equal(t, "Residential Tower B", found.Name)
	assert.Equal(t, "Calle Mayor 12, Madrid", found.Address)
	assert.Equal(t, "Carlos Pérez", found.ManagerName)
	assert.Equal(t, "+34 699 111 222", found.Phone)
	assert.Equal(t, "Test project description", found.Description)
	assert.Equal(t, client.ID, found.ClientID)
	assert.False(t, found.CreatedAt.IsZero())
	assert.False(t, found.UpdatedAt.IsZero())
}

// =============================================================================
// ListByClientID
// =============================================================================

func TestPostgresProjectRepository_ListByClientID(t *testing.T) {
	projectRepo, clientRepo, userRepo := setupProjectRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)

	p1, _ := entities.NewProject(entities.CreateProjectParams{Name: "Project A", ClientID: client.ID})
	p2, _ := entities.NewProject(entities.CreateProjectParams{Name: "Project B", ClientID: client.ID})
	require.NoError(t, projectRepo.Create(ctx, p1))
	require.NoError(t, projectRepo.Create(ctx, p2))

	pg, _ := query.NewPageRequest(1, 20)
	projects, total, err := projectRepo.ListByClientID(ctx, client.ID, projectListQuery(pg))
	require.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Equal(t, int64(2), total)
}

func TestPostgresProjectRepository_ListByClientID_SearchSortAndPagination(t *testing.T) {
	projectRepo, clientRepo, userRepo := setupProjectRepo(t)
	ctx := context.Background()
	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)

	projects := []entities.CreateProjectParams{
		{Name: "Beta Tower", ClientID: client.ID},
		{Name: "Alpha Job", Address: "Tower Street", ClientID: client.ID},
		{Name: "Gamma Job", ManagerName: "Tower Lead", ClientID: client.ID},
		{Name: "Unrelated", ClientID: client.ID},
	}
	for _, params := range projects {
		project, err := entities.NewProject(params)
		require.NoError(t, err)
		require.NoError(t, projectRepo.Create(ctx, project))
	}

	pg, _ := query.NewPageRequest(1, 2)
	opts := query.ProjectListQuery{ListQuery: query.ListQuery{
		Page:   pg,
		Search: "tower",
		Sort:   &query.Sort{Field: "name", Direction: query.SortAscending},
	}}
	firstPage, total, err := projectRepo.ListByClientID(ctx, client.ID, opts)
	require.NoError(t, err)
	require.Len(t, firstPage, 2)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, "Alpha Job", firstPage[0].Name)
	assert.Equal(t, "Beta Tower", firstPage[1].Name)

	pg, _ = query.NewPageRequest(2, 2)
	opts.Page = pg
	secondPage, total, err := projectRepo.ListByClientID(ctx, client.ID, opts)
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, "Gamma Job", secondPage[0].Name)
}

func TestPostgresProjectRepository_ListByClientID_Empty(t *testing.T) {
	projectRepo, clientRepo, userRepo := setupProjectRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)

	pg, _ := query.NewPageRequest(1, 20)
	projects, total, err := projectRepo.ListByClientID(ctx, client.ID, projectListQuery(pg))
	require.NoError(t, err)
	assert.Empty(t, projects)
	assert.Equal(t, int64(0), total)
}

func TestPostgresProjectRepository_ListByClientID_DoesNotReturnOtherClientsProjects(t *testing.T) {
	projectRepo, clientRepo, userRepo := setupProjectRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client1 := createTestClient(t, clientRepo, user.ID)
	client2 := createTestClient(t, clientRepo, user.ID)

	pA, _ := entities.NewProject(entities.CreateProjectParams{Name: "Shared Name", ClientID: client1.ID})
	pB, _ := entities.NewProject(entities.CreateProjectParams{Name: "Shared Name", ClientID: client2.ID})
	require.NoError(t, projectRepo.Create(ctx, pA))
	require.NoError(t, projectRepo.Create(ctx, pB))

	// Client 1 should only see their own project
	pg, _ := query.NewPageRequest(1, 20)
	projects1, _, err := projectRepo.ListByClientID(ctx, client1.ID, projectListQuery(pg))
	require.NoError(t, err)
	assert.Len(t, projects1, 1)
	assert.Equal(t, client1.ID, projects1[0].ClientID)

	// Client 2 should only see their own project
	projects2, _, err := projectRepo.ListByClientID(ctx, client2.ID, projectListQuery(pg))
	require.NoError(t, err)
	assert.Len(t, projects2, 1)
	assert.Equal(t, client2.ID, projects2[0].ClientID)
}

// =============================================================================
// Update
// =============================================================================

func TestPostgresProjectRepository_Update(t *testing.T) {
	projectRepo, clientRepo, userRepo := setupProjectRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project, _ := entities.NewProject(entities.CreateProjectParams{Name: "Old Name", ClientID: client.ID})
	require.NoError(t, projectRepo.Create(ctx, project))

	require.NoError(t, project.SetName("New Name"))
	require.NoError(t, project.SetAddress("Avenida de la Constitución 1"))
	require.NoError(t, project.SetManagerName("Ana García"))
	require.NoError(t, project.SetPhone("+34 999 888 777"))
	require.NoError(t, project.SetDescription("Updated description"))
	err := projectRepo.Update(ctx, project)
	require.NoError(t, err)

	found, err := projectRepo.GetByID(ctx, project.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", found.Name)
	assert.Equal(t, "Avenida de la Constitución 1", found.Address)
	assert.Equal(t, "Ana García", found.ManagerName)
	assert.Equal(t, "+34 999 888 777", found.Phone)
	assert.Equal(t, "Updated description", found.Description)
}

// =============================================================================
// Delete
// =============================================================================

func TestPostgresProjectRepository_Delete(t *testing.T) {
	projectRepo, clientRepo, userRepo := setupProjectRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project, _ := entities.NewProject(entities.CreateProjectParams{Name: "To Delete", ClientID: client.ID})
	require.NoError(t, projectRepo.Create(ctx, project))

	err := projectRepo.Delete(ctx, project.ID)
	require.NoError(t, err)

	found, err := projectRepo.GetByID(ctx, project.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

// =============================================================================
// GetByID — Not Found
// =============================================================================

func TestPostgresProjectRepository_GetByID_NotFound(t *testing.T) {
	projectRepo, _, _ := setupProjectRepo(t)
	ctx := context.Background()

	found, err := projectRepo.GetByID(ctx, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, found)
}

// =============================================================================
// Create — FK violation (non-existing client)
// =============================================================================

func TestPostgresProjectRepository_Create_WithInvalidClientID_Fails(t *testing.T) {
	projectRepo, _, _ := setupProjectRepo(t)
	ctx := context.Background()

	project, _ := entities.NewProject(entities.CreateProjectParams{Name: "Orphan Project", ClientID: uuid.New()})
	err := projectRepo.Create(ctx, project)

	assert.Error(t, err, "creating a project with a non-existing client_id should fail due to FK constraint")
}

// =============================================================================
// Two Clients Same Project Name — Both succeed
// =============================================================================

func TestPostgresProjectRepository_TwoClients_SameProjectName_BothSucceed(t *testing.T) {
	projectRepo, clientRepo, userRepo := setupProjectRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client1 := createTestClient(t, clientRepo, user.ID)
	client2 := createTestClient(t, clientRepo, user.ID)

	pA, _ := entities.NewProject(entities.CreateProjectParams{Name: "Same Name", ClientID: client1.ID})
	pB, _ := entities.NewProject(entities.CreateProjectParams{Name: "Same Name", ClientID: client2.ID})

	require.NoError(t, projectRepo.Create(ctx, pA))
	require.NoError(t, projectRepo.Create(ctx, pB))

	foundA, err := projectRepo.GetByID(ctx, pA.ID)
	require.NoError(t, err)
	assert.Equal(t, "Same Name", foundA.Name)
	assert.Equal(t, client1.ID, foundA.ClientID)

	foundB, err := projectRepo.GetByID(ctx, pB.ID)
	require.NoError(t, err)
	assert.Equal(t, "Same Name", foundB.Name)
	assert.Equal(t, client2.ID, foundB.ClientID)

	assert.NotEqual(t, foundA.ID, foundB.ID, "they are different projects even though they have the same name")
}

// =============================================================================
// Mapper — Data integrity
// =============================================================================

func TestPostgresProjectRepository_Mapper_PreservesAllFields(t *testing.T) {
	projectRepo, clientRepo, userRepo := setupProjectRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	original, _ := entities.NewProject(newProjectParams("Full Data Project", client.ID))
	require.NoError(t, projectRepo.Create(ctx, original))

	found, err := projectRepo.GetByID(ctx, original.ID)
	require.NoError(t, err)

	assert.Equal(t, original.ID, found.ID)
	assert.Equal(t, original.Name, found.Name)
	assert.Equal(t, original.Address, found.Address)
	assert.Equal(t, original.ManagerName, found.ManagerName)
	assert.Equal(t, original.Phone, found.Phone)
	assert.Equal(t, original.Description, found.Description)
	assert.Equal(t, original.ClientID, found.ClientID)
	assert.WithinDuration(t, original.CreatedAt, found.CreatedAt, 1_000_000_000)
	assert.WithinDuration(t, original.UpdatedAt, found.UpdatedAt, 1_000_000_000)
}

// =============================================================================
// Cascade Delete — Deleting client deletes their projects
// =============================================================================

func TestPostgresProjectRepository_CascadeDelete_ClientDeletion(t *testing.T) {
	projectRepo, clientRepo, userRepo := setupProjectRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client := createTestClient(t, clientRepo, user.ID)
	project, _ := entities.NewProject(entities.CreateProjectParams{Name: "Will Be Orphaned", ClientID: client.ID})
	require.NoError(t, projectRepo.Create(ctx, project))

	// Delete the client directly via DB (simulating cascade)
	db := helpers.SetupTestDB(t)
	err := db.Exec("DELETE FROM clients WHERE id = ?", client.ID).Error
	require.NoError(t, err)

	// The project should be gone too (CASCADE)
	found, err := projectRepo.GetByID(ctx, project.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}
