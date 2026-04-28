package persistence_test

import (
	"context"
	"testing"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"
	"ductifact/internal/infrastructure/adapters/outbound/persistence"
	"ductifact/test/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupOneTimeTokenRepo(
	t *testing.T,
) (*persistence.PostgresOneTimeTokenRepository, *persistence.PostgresUserRepository) {
	db := helpers.SetupTestDB(t)
	helpers.CleanDB(t, db)
	return persistence.NewPostgresOneTimeTokenRepository(db), persistence.NewPostgresUserRepository(db)
}

// createTokenTestUser creates and persists a user required as FK for tokens.
func createTokenTestUser(t *testing.T, userRepo *persistence.PostgresUserRepository) *entities.User {
	t.Helper()
	user, err := entities.NewUser(entities.CreateUserParams{
		Name: "Token User", Email: "token@example.com", Password: "securepass123", Locale: "en",
	})
	require.NoError(t, err)
	require.NoError(t, userRepo.Create(context.Background(), user))
	return user
}

func TestOneTimeTokenRepository_Create_And_GetByToken(t *testing.T) {
	repo, userRepo := setupOneTimeTokenRepo(t)
	ctx := context.Background()
	user := createTokenTestUser(t, userRepo)

	// Create token
	token, err := entities.NewOneTimeToken(user.ID, entities.TokenTypeEmailVerification, 24*time.Hour)
	require.NoError(t, err)

	err = repo.Create(ctx, token)
	require.NoError(t, err)

	// Retrieve by token string + type
	found, err := repo.GetByToken(ctx, token.Token, entities.TokenTypeEmailVerification)
	require.NoError(t, err)

	assert.Equal(t, token.ID, found.ID)
	assert.Equal(t, user.ID, found.UserID)
	assert.Equal(t, token.Token, found.Token)
	assert.Equal(t, entities.TokenTypeEmailVerification, found.Type)
	assert.WithinDuration(t, token.ExpiresAt, found.ExpiresAt, time.Second)
}

func TestOneTimeTokenRepository_GetByToken_WrongType_ReturnsNotFound(t *testing.T) {
	repo, userRepo := setupOneTimeTokenRepo(t)
	ctx := context.Background()
	user := createTokenTestUser(t, userRepo)

	token, _ := entities.NewOneTimeToken(user.ID, entities.TokenTypeEmailVerification, 24*time.Hour)
	require.NoError(t, repo.Create(ctx, token))

	// Search with wrong type
	_, err := repo.GetByToken(ctx, token.Token, entities.TokenTypePasswordReset)

	assert.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestOneTimeTokenRepository_GetByToken_NotFound(t *testing.T) {
	repo, _ := setupOneTimeTokenRepo(t)
	ctx := context.Background()

	_, err := repo.GetByToken(ctx, "nonexistent-token", entities.TokenTypeEmailVerification)

	assert.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestOneTimeTokenRepository_DeleteByUserIDAndType(t *testing.T) {
	repo, userRepo := setupOneTimeTokenRepo(t)
	ctx := context.Background()
	user := createTokenTestUser(t, userRepo)

	// Create two tokens of the same type
	token1, _ := entities.NewOneTimeToken(user.ID, entities.TokenTypeEmailVerification, 24*time.Hour)
	token2, _ := entities.NewOneTimeToken(user.ID, entities.TokenTypeEmailVerification, 24*time.Hour)
	require.NoError(t, repo.Create(ctx, token1))
	require.NoError(t, repo.Create(ctx, token2))

	// Delete all email verification tokens for the user
	err := repo.DeleteByUserIDAndType(ctx, user.ID, entities.TokenTypeEmailVerification)
	require.NoError(t, err)

	// Both should be gone
	_, err = repo.GetByToken(ctx, token1.Token, entities.TokenTypeEmailVerification)
	assert.ErrorIs(t, err, repositories.ErrNotFound)

	_, err = repo.GetByToken(ctx, token2.Token, entities.TokenTypeEmailVerification)
	assert.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestOneTimeTokenRepository_DeleteByUserIDAndType_DoesNotAffectOtherTypes(t *testing.T) {
	repo, userRepo := setupOneTimeTokenRepo(t)
	ctx := context.Background()
	user := createTokenTestUser(t, userRepo)

	// Create one token of each type
	emailToken, _ := entities.NewOneTimeToken(user.ID, entities.TokenTypeEmailVerification, 24*time.Hour)
	resetToken, _ := entities.NewOneTimeToken(user.ID, entities.TokenTypePasswordReset, time.Hour)
	require.NoError(t, repo.Create(ctx, emailToken))
	require.NoError(t, repo.Create(ctx, resetToken))

	// Delete only email verification tokens
	err := repo.DeleteByUserIDAndType(ctx, user.ID, entities.TokenTypeEmailVerification)
	require.NoError(t, err)

	// Email token gone
	_, err = repo.GetByToken(ctx, emailToken.Token, entities.TokenTypeEmailVerification)
	assert.ErrorIs(t, err, repositories.ErrNotFound)

	// Password reset token still exists
	found, err := repo.GetByToken(ctx, resetToken.Token, entities.TokenTypePasswordReset)
	require.NoError(t, err)
	assert.Equal(t, resetToken.ID, found.ID)
}

func TestOneTimeTokenRepository_DeleteByUserIDAndType_DoesNotAffectOtherUsers(t *testing.T) {
	repo, userRepo := setupOneTimeTokenRepo(t)
	ctx := context.Background()
	user1 := createTokenTestUser(t, userRepo)

	// Create second user manually
	user2, _ := entities.NewUser(entities.CreateUserParams{
		Name: "Other User", Email: "other@example.com", Password: "securepass123", Locale: "en",
	})
	require.NoError(t, userRepo.Create(ctx, user2))

	token1, _ := entities.NewOneTimeToken(user1.ID, entities.TokenTypeEmailVerification, 24*time.Hour)
	token2, _ := entities.NewOneTimeToken(user2.ID, entities.TokenTypeEmailVerification, 24*time.Hour)
	require.NoError(t, repo.Create(ctx, token1))
	require.NoError(t, repo.Create(ctx, token2))

	// Delete user1's tokens only
	require.NoError(t, repo.DeleteByUserIDAndType(ctx, user1.ID, entities.TokenTypeEmailVerification))

	// User1 token gone
	_, err := repo.GetByToken(ctx, token1.Token, entities.TokenTypeEmailVerification)
	assert.ErrorIs(t, err, repositories.ErrNotFound)

	// User2 token still exists
	found, err := repo.GetByToken(ctx, token2.Token, entities.TokenTypeEmailVerification)
	require.NoError(t, err)
	assert.Equal(t, user2.ID, found.UserID)
	_ = found
}

func TestOneTimeTokenRepository_DeleteByUserIDAndType_NonexistentUser_NoError(t *testing.T) {
	repo, _ := setupOneTimeTokenRepo(t)
	ctx := context.Background()

	err := repo.DeleteByUserIDAndType(ctx, uuid.New(), entities.TokenTypeEmailVerification)

	assert.NoError(t, err)
}
