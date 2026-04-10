package persistence_test

import (
	"context"
	"testing"

	"ductifact/internal/domain/entities"
	"ductifact/internal/infrastructure/adapters/outbound/persistence"
	"ductifact/test/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRepo creates a repository with a clean DB for each test.
func setupRepo(t *testing.T) *persistence.PostgresUserRepository {
	db := helpers.SetupTestDB(t)
	helpers.CleanDB(t, db)
	return persistence.NewPostgresUserRepository(db)
}

// =============================================================================
// Create + GetByID
// =============================================================================

func TestPostgresUserRepository_Create_And_GetByID(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	// Create a valid user using the domain constructor
	user, err := entities.NewUser("Juan", "juan@example.com", "securepass123")
	require.NoError(t, err)

	// CREATE
	err = repo.Create(ctx, user)
	require.NoError(t, err)

	// GET BY ID
	found, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)

	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "Juan", found.Name)
	assert.Equal(t, "juan@example.com", found.Email)
	assert.False(t, found.CreatedAt.IsZero())
	assert.False(t, found.UpdatedAt.IsZero())
}

// =============================================================================
// GetByEmail
// =============================================================================

func TestPostgresUserRepository_GetByEmail(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	user, _ := entities.NewUser("Juan", "juan@example.com", "securepass123")
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Existing email
	found, err := repo.GetByEmail(ctx, "juan@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "Juan", found.Name)

	// Non-existing email
	notFound, err := repo.GetByEmail(ctx, "noexiste@example.com")
	assert.Error(t, err)
	assert.Nil(t, notFound)
}

// =============================================================================
// Update
// =============================================================================

func TestPostgresUserRepository_Update(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	user, _ := entities.NewUser("Juan", "juan@example.com", "securepass123")
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Modify the user
	user.Name = "Pedro"
	user.Email = "pedro@example.com"

	err = repo.Update(ctx, user)
	require.NoError(t, err)

	// Verify the update was persisted
	found, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Pedro", found.Name)
	assert.Equal(t, "pedro@example.com", found.Email)
}

// =============================================================================
// GetByID — Not Found
// =============================================================================

func TestPostgresUserRepository_GetByID_NotFound(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	found, err := repo.GetByID(ctx, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, found)
}

// =============================================================================
// Create — Duplicate Email (DB UNIQUE constraint)
// =============================================================================

func TestPostgresUserRepository_Create_DuplicateEmail_Fails(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	user1, _ := entities.NewUser("Juan", "same@example.com", "securepass123")
	err := repo.Create(ctx, user1)
	require.NoError(t, err)

	user2, _ := entities.NewUser("Pedro", "same@example.com", "securepass123")
	err = repo.Create(ctx, user2)

	// The DB must reject the duplicate email via UNIQUE constraint
	assert.Error(t, err)
}

// =============================================================================
// Create — Multiple Users
// =============================================================================

func TestPostgresUserRepository_Create_MultipleUsers(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	user1, _ := entities.NewUser("Juan", "juan@example.com", "securepass123")
	user2, _ := entities.NewUser("Pedro", "pedro@example.com", "securepass123")
	user3, _ := entities.NewUser("Maria", "maria@example.com", "securepass123")

	require.NoError(t, repo.Create(ctx, user1))
	require.NoError(t, repo.Create(ctx, user2))
	require.NoError(t, repo.Create(ctx, user3))

	// Verify each user can be retrieved
	found1, err := repo.GetByID(ctx, user1.ID)
	require.NoError(t, err)
	assert.Equal(t, "Juan", found1.Name)

	found2, err := repo.GetByID(ctx, user2.ID)
	require.NoError(t, err)
	assert.Equal(t, "Pedro", found2.Name)

	found3, err := repo.GetByID(ctx, user3.ID)
	require.NoError(t, err)
	assert.Equal(t, "Maria", found3.Name)
}

// =============================================================================
// Mapper — Data integrity (Create → GetByID preserves all fields)
// =============================================================================

func TestPostgresUserRepository_Mapper_PreservesAllFields(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	original, _ := entities.NewUser("Juan García", "juan.garcia@example.com", "securepass123")
	err := repo.Create(ctx, original)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, original.ID)
	require.NoError(t, err)

	// Verify the mappers (toUserModel → DB → toUserEntity) don't lose data
	assert.Equal(t, original.ID, found.ID)
	assert.Equal(t, original.Name, found.Name)
	assert.Equal(t, original.Email, found.Email)
	// Timestamps: compare with 1-second tolerance (DB may truncate microseconds)
	assert.WithinDuration(t, original.CreatedAt, found.CreatedAt, 1_000_000_000)
	assert.WithinDuration(t, original.UpdatedAt, found.UpdatedAt, 1_000_000_000)
}

// =============================================================================
// Soft Delete — Re-register with same email after soft delete
// =============================================================================

func TestPostgresUserRepository_Create_SameEmail_AfterSoftDelete_Succeeds(t *testing.T) {
	db := helpers.SetupTestDB(t)
	helpers.CleanDB(t, db)
	repo := persistence.NewPostgresUserRepository(db)
	ctx := context.Background()

	// 1. Create a user
	user1, _ := entities.NewUser("Juan", "reuse@example.com", "securepass123")
	err := repo.Create(ctx, user1)
	require.NoError(t, err)

	// 2. Soft-delete it via GORM (sets deleted_at = NOW())
	err = db.Delete(&persistence.UserModel{}, "id = ?", user1.ID).Error
	require.NoError(t, err)

	// 3. Create a new user with the SAME email — must succeed thanks to partial unique index
	user2, _ := entities.NewUser("Juan Nuevo", "reuse@example.com", "securepass123")
	err = repo.Create(ctx, user2)
	require.NoError(t, err, "should allow re-registration after soft delete")

	// 4. Verify the new user is retrievable
	found, err := repo.GetByEmail(ctx, "reuse@example.com")
	require.NoError(t, err)
	assert.Equal(t, user2.ID, found.ID)
	assert.Equal(t, "Juan Nuevo", found.Name)
}
