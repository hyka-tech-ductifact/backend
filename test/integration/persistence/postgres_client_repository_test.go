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

// setupClientRepo creates both user and client repos with a clean DB.
func setupClientRepo(t *testing.T) (*persistence.PostgresClientRepository, *persistence.PostgresUserRepository) {
	db := helpers.SetupTestDB(t)
	helpers.CleanDB(t, db)
	return persistence.NewPostgresClientRepository(db), persistence.NewPostgresUserRepository(db)
}

// createTestUser is a helper that creates and persists a user for FK tests.
func createTestUser(t *testing.T, userRepo *persistence.PostgresUserRepository) *entities.User {
	user, err := entities.NewUser("Test User", "testuser_"+uuid.New().String()[:8]+"@example.com", "securepass123")
	require.NoError(t, err)
	require.NoError(t, userRepo.Create(context.Background(), user))
	return user
}

// newClientParams is a helper that returns CreateClientParams with all fields populated.
func newClientParams(name string, userID uuid.UUID) entities.CreateClientParams {
	return entities.CreateClientParams{
		Name:        name,
		Phone:       "+34 600 111 222",
		Email:       "client@example.com",
		Description: "Test client description",
		UserID:      userID,
	}
}

// =============================================================================
// Create + GetByID
// =============================================================================

func TestPostgresClientRepository_Create_And_GetByID(t *testing.T) {
	clientRepo, userRepo := setupClientRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	client, err := entities.NewClient(newClientParams("Acme Corp", user.ID))
	require.NoError(t, err)

	err = clientRepo.Create(ctx, client)
	require.NoError(t, err)

	found, err := clientRepo.GetByID(ctx, client.ID)
	require.NoError(t, err)

	assert.Equal(t, client.ID, found.ID)
	assert.Equal(t, "Acme Corp", found.Name)
	assert.Equal(t, "+34 600 111 222", found.Phone)
	assert.Equal(t, "client@example.com", found.Email)
	assert.Equal(t, "Test client description", found.Description)
	assert.Equal(t, user.ID, found.UserID)
	assert.False(t, found.CreatedAt.IsZero())
	assert.False(t, found.UpdatedAt.IsZero())
}

// =============================================================================
// ListByUserID
// =============================================================================

func TestPostgresClientRepository_ListByUserID(t *testing.T) {
	clientRepo, userRepo := setupClientRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	client1, _ := entities.NewClient(entities.CreateClientParams{Name: "Client A", UserID: user.ID})
	client2, _ := entities.NewClient(entities.CreateClientParams{Name: "Client B", UserID: user.ID})
	require.NoError(t, clientRepo.Create(ctx, client1))
	require.NoError(t, clientRepo.Create(ctx, client2))

	pg, _ := pagination.NewPagination(1, 20)
	clients, total, err := clientRepo.ListByUserID(ctx, user.ID, pg)
	require.NoError(t, err)
	assert.Len(t, clients, 2)
	assert.Equal(t, int64(2), total)
}

func TestPostgresClientRepository_ListByUserID_Empty(t *testing.T) {
	clientRepo, userRepo := setupClientRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	pg, _ := pagination.NewPagination(1, 20)
	clients, total, err := clientRepo.ListByUserID(ctx, user.ID, pg)
	require.NoError(t, err)
	assert.Empty(t, clients)
	assert.Equal(t, int64(0), total)
}

func TestPostgresClientRepository_ListByUserID_DoesNotReturnOtherUsersClients(t *testing.T) {
	clientRepo, userRepo := setupClientRepo(t)
	ctx := context.Background()

	user1 := createTestUser(t, userRepo)
	user2 := createTestUser(t, userRepo)

	clientA, _ := entities.NewClient(entities.CreateClientParams{Name: "Shared Name", UserID: user1.ID})
	clientB, _ := entities.NewClient(entities.CreateClientParams{Name: "Shared Name", UserID: user2.ID})
	require.NoError(t, clientRepo.Create(ctx, clientA))
	require.NoError(t, clientRepo.Create(ctx, clientB))

	// User 1 should only see their own client
	pg, _ := pagination.NewPagination(1, 20)
	clients1, _, err := clientRepo.ListByUserID(ctx, user1.ID, pg)
	require.NoError(t, err)
	assert.Len(t, clients1, 1)
	assert.Equal(t, user1.ID, clients1[0].UserID)

	// User 2 should only see their own client
	clients2, _, err := clientRepo.ListByUserID(ctx, user2.ID, pg)
	require.NoError(t, err)
	assert.Len(t, clients2, 1)
	assert.Equal(t, user2.ID, clients2[0].UserID)
}

// =============================================================================
// Update
// =============================================================================

func TestPostgresClientRepository_Update(t *testing.T) {
	clientRepo, userRepo := setupClientRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client, _ := entities.NewClient(entities.CreateClientParams{Name: "Old Name", UserID: user.ID})
	require.NoError(t, clientRepo.Create(ctx, client))

	require.NoError(t, client.SetName("New Name"))
	require.NoError(t, client.SetPhone("+34 999 888 777"))
	require.NoError(t, client.SetEmail("updated@example.com"))
	require.NoError(t, client.SetDescription("Updated description"))
	err := clientRepo.Update(ctx, client)
	require.NoError(t, err)

	found, err := clientRepo.GetByID(ctx, client.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", found.Name)
	assert.Equal(t, "+34 999 888 777", found.Phone)
	assert.Equal(t, "updated@example.com", found.Email)
	assert.Equal(t, "Updated description", found.Description)
}

// =============================================================================
// Delete
// =============================================================================

func TestPostgresClientRepository_Delete(t *testing.T) {
	clientRepo, userRepo := setupClientRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client, _ := entities.NewClient(entities.CreateClientParams{Name: "To Delete", UserID: user.ID})
	require.NoError(t, clientRepo.Create(ctx, client))

	err := clientRepo.Delete(ctx, client.ID)
	require.NoError(t, err)

	found, err := clientRepo.GetByID(ctx, client.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

// =============================================================================
// GetByID — Not Found
// =============================================================================

func TestPostgresClientRepository_GetByID_NotFound(t *testing.T) {
	clientRepo, _ := setupClientRepo(t)
	ctx := context.Background()

	found, err := clientRepo.GetByID(ctx, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, found)
}

// =============================================================================
// Create — FK violation (non-existing user)
// =============================================================================

func TestPostgresClientRepository_Create_WithInvalidUserID_Fails(t *testing.T) {
	clientRepo, _ := setupClientRepo(t)
	ctx := context.Background()

	client, _ := entities.NewClient(entities.CreateClientParams{Name: "Orphan Client", UserID: uuid.New()})
	err := clientRepo.Create(ctx, client)

	assert.Error(t, err, "creating a client with a non-existing user_id should fail due to FK constraint")
}

// =============================================================================
// Two Users Same Client Name — Both succeed
// =============================================================================

func TestPostgresClientRepository_TwoUsers_SameClientName_BothSucceed(t *testing.T) {
	clientRepo, userRepo := setupClientRepo(t)
	ctx := context.Background()

	user1 := createTestUser(t, userRepo)
	user2 := createTestUser(t, userRepo)

	clientA, _ := entities.NewClient(entities.CreateClientParams{Name: "Same Name", UserID: user1.ID})
	clientB, _ := entities.NewClient(entities.CreateClientParams{Name: "Same Name", UserID: user2.ID})

	require.NoError(t, clientRepo.Create(ctx, clientA))
	require.NoError(t, clientRepo.Create(ctx, clientB))

	foundA, err := clientRepo.GetByID(ctx, clientA.ID)
	require.NoError(t, err)
	assert.Equal(t, "Same Name", foundA.Name)
	assert.Equal(t, user1.ID, foundA.UserID)

	foundB, err := clientRepo.GetByID(ctx, clientB.ID)
	require.NoError(t, err)
	assert.Equal(t, "Same Name", foundB.Name)
	assert.Equal(t, user2.ID, foundB.UserID)

	assert.NotEqual(t, foundA.ID, foundB.ID, "they are different clients even though they have the same name")
}

// =============================================================================
// Mapper — Data integrity
// =============================================================================

func TestPostgresClientRepository_Mapper_PreservesAllFields(t *testing.T) {
	clientRepo, userRepo := setupClientRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	original, _ := entities.NewClient(newClientParams("Full Data Client", user.ID))
	require.NoError(t, clientRepo.Create(ctx, original))

	found, err := clientRepo.GetByID(ctx, original.ID)
	require.NoError(t, err)

	assert.Equal(t, original.ID, found.ID)
	assert.Equal(t, original.Name, found.Name)
	assert.Equal(t, original.Phone, found.Phone)
	assert.Equal(t, original.Email, found.Email)
	assert.Equal(t, original.Description, found.Description)
	assert.Equal(t, original.UserID, found.UserID)
	assert.WithinDuration(t, original.CreatedAt, found.CreatedAt, 1_000_000_000)
	assert.WithinDuration(t, original.UpdatedAt, found.UpdatedAt, 1_000_000_000)
}

// =============================================================================
// Cascade Delete — Deleting user deletes their clients
// =============================================================================

func TestPostgresClientRepository_CascadeDelete_UserDeletion(t *testing.T) {
	clientRepo, userRepo := setupClientRepo(t)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	client, _ := entities.NewClient(entities.CreateClientParams{Name: "Will Be Orphaned", UserID: user.ID})
	require.NoError(t, clientRepo.Create(ctx, client))

	// Delete the user directly via DB (simulating cascade)
	db := helpers.SetupTestDB(t)
	err := db.Exec("DELETE FROM users WHERE id = ?", user.ID).Error
	require.NoError(t, err)

	// The client should be gone too (CASCADE)
	found, err := clientRepo.GetByID(ctx, client.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}
