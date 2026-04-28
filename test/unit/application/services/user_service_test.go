package services_test

import (
	"context"
	"errors"
	"testing"

	"ductifact/internal/application/services"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"
	"ductifact/internal/domain/valueobjects"
	"ductifact/test/unit/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// GetUserByID
// =============================================================================

func TestGetUserByID_WithExistingUser_ReturnsUser(t *testing.T) {
	user := newTestUser()
	svc := services.NewUserService(userRepoReturning(user), &mocks.MockClientRepository{})

	result, err := svc.GetUserByID(context.Background(), user.ID)

	require.NoError(t, err)
	assert.Equal(t, user.Name, result.Name)
	assert.Equal(t, user.Email, result.Email)
}

func TestGetUserByID_WithNonExistingUser_ReturnsError(t *testing.T) {
	svc := services.NewUserService(userRepoReturning(nil), &mocks.MockClientRepository{})

	result, err := svc.GetUserByID(context.Background(), uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrUserNotFound)
}

// =============================================================================
// UpdateUser
// =============================================================================

func TestUpdateUser_WithNewName_UpdatesOnlyName(t *testing.T) {
	user := newTestUser()
	svc := services.NewUserService(userRepoReturning(user), &mocks.MockClientRepository{})

	result, err := svc.UpdateUser(context.Background(), user.ID, strPtr("Pedro"), nil, nil)

	require.NoError(t, err)
	assert.Equal(t, "Pedro", result.Name)
	assert.Equal(t, user.Email, result.Email)
}

func TestUpdateUser_WithNewEmail_UpdatesOnlyEmail(t *testing.T) {
	user := newTestUser()
	repo := userRepoReturning(user)
	repo.GetByEmailFn = func(ctx context.Context, email string) (*entities.User, error) {
		return nil, repositories.ErrNotFound
	}
	svc := services.NewUserService(repo, &mocks.MockClientRepository{})

	result, err := svc.UpdateUser(context.Background(), user.ID, nil, strPtr("pedro@example.com"), nil)

	require.NoError(t, err)
	assert.Equal(t, "Juan", result.Name)
	assert.Equal(t, "pedro@example.com", result.Email)
}

func TestUpdateUser_WithDuplicateEmail_ReturnsError(t *testing.T) {
	user := newTestUser()
	repo := userRepoReturning(user)
	repo.GetByEmailFn = func(ctx context.Context, email string) (*entities.User, error) {
		return &entities.User{ID: uuid.New(), Email: email}, nil
	}
	svc := services.NewUserService(repo, &mocks.MockClientRepository{})

	result, err := svc.UpdateUser(context.Background(), user.ID, nil, strPtr("taken@example.com"), nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrEmailAlreadyInUse)
}

func TestUpdateUser_WithSameEmail_DoesNotCheckUniqueness(t *testing.T) {
	user := newTestUser()
	repo := userRepoReturning(user)
	getByEmailCalled := false
	repo.GetByEmailFn = func(ctx context.Context, email string) (*entities.User, error) {
		getByEmailCalled = true
		return nil, nil
	}
	svc := services.NewUserService(repo, &mocks.MockClientRepository{})

	result, err := svc.UpdateUser(context.Background(), user.ID, nil, strPtr(user.Email), nil)

	require.NoError(t, err)
	assert.Equal(t, user.Email, result.Email)
	assert.False(t, getByEmailCalled, "GetByEmail should not be called when email does not change")
}

func TestUpdateUser_WithNonExistingUser_ReturnsError(t *testing.T) {
	svc := services.NewUserService(userRepoReturning(nil), &mocks.MockClientRepository{})

	result, err := svc.UpdateUser(context.Background(), uuid.New(), strPtr("Pedro"), nil, nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrUserNotFound)
}

func TestUpdateUser_WhenRepoFails_ReturnsError(t *testing.T) {
	user := newTestUser()
	repo := userRepoReturning(user)
	repo.UpdateFn = func(ctx context.Context, u *entities.User) error {
		return errors.New("db write failed")
	}
	svc := services.NewUserService(repo, &mocks.MockClientRepository{})

	result, err := svc.UpdateUser(context.Background(), user.ID, strPtr("Pedro"), nil, nil)

	assert.Nil(t, result)
	assert.EqualError(t, err, "db write failed")
}

func TestUpdateUser_UpdatesTimestamp(t *testing.T) {
	user := newTestUser()
	oldTime := user.UpdatedAt
	svc := services.NewUserService(userRepoReturning(user), &mocks.MockClientRepository{})

	result, err := svc.UpdateUser(context.Background(), user.ID, strPtr("Pedro"), nil, nil)

	require.NoError(t, err)
	assert.True(t, result.UpdatedAt.After(oldTime), "UpdatedAt must be newer than the original")
}

func TestUpdateUser_WithNoChanges_SkipsPersistence(t *testing.T) {
	user := newTestUser()
	repo := userRepoReturning(user)
	updateCalled := false
	repo.UpdateFn = func(ctx context.Context, u *entities.User) error {
		updateCalled = true
		return nil
	}
	svc := services.NewUserService(repo, &mocks.MockClientRepository{})

	_, err := svc.UpdateUser(context.Background(), user.ID, nil, nil, nil)

	require.NoError(t, err)
	assert.False(t, updateCalled, "Update should not be called when no fields change")
}

func TestUpdateUser_WithEmptyName_ReturnsError(t *testing.T) {
	user := newTestUser()
	svc := services.NewUserService(userRepoReturning(user), &mocks.MockClientRepository{})

	result, err := svc.UpdateUser(context.Background(), user.ID, strPtr(""), nil, nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, entities.ErrEmptyUserName)
}

func TestUpdateUser_WithInvalidEmailFormat_ReturnsError(t *testing.T) {
	user := newTestUser()
	svc := services.NewUserService(userRepoReturning(user), &mocks.MockClientRepository{})

	result, err := svc.UpdateUser(context.Background(), user.ID, nil, strPtr("not-an-email"), nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, valueobjects.ErrInvalidEmail)
}

func TestUpdateUser_WhenGetByEmailFails_ReturnsError(t *testing.T) {
	user := newTestUser()
	repo := userRepoReturning(user)
	repo.GetByEmailFn = func(ctx context.Context, email string) (*entities.User, error) {
		return nil, errors.New("db connection lost")
	}
	svc := services.NewUserService(repo, &mocks.MockClientRepository{})

	result, err := svc.UpdateUser(context.Background(), user.ID, nil, strPtr("pedro@example.com"), nil)

	assert.Nil(t, result)
	assert.EqualError(t, err, "db connection lost")
}

// =============================================================================
// UpdateUser — Locale
// =============================================================================

func TestUpdateUser_WithValidLocale_UpdatesLocale(t *testing.T) {
	user := newTestUser()
	svc := services.NewUserService(userRepoReturning(user), &mocks.MockClientRepository{})

	result, err := svc.UpdateUser(context.Background(), user.ID, nil, nil, strPtr("es"))

	require.NoError(t, err)
	assert.Equal(t, "es", result.Locale)
}

func TestUpdateUser_WithInvalidLocale_ReturnsError(t *testing.T) {
	user := newTestUser()
	svc := services.NewUserService(userRepoReturning(user), &mocks.MockClientRepository{})

	result, err := svc.UpdateUser(context.Background(), user.ID, nil, nil, strPtr("fr"))

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid locale")
}

// =============================================================================
// DeleteUser
// =============================================================================

func TestDeleteUser_WithNoClients_DeletesSuccessfully(t *testing.T) {
	user := newTestUser()
	deleteCalled := false
	repo := userRepoReturning(user)
	repo.DeleteFn = func(ctx context.Context, id uuid.UUID) error {
		deleteCalled = true
		assert.Equal(t, user.ID, id)
		return nil
	}
	svc := services.NewUserService(repo, &mocks.MockClientRepository{})

	err := svc.DeleteUser(context.Background(), user.ID, false)

	require.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestDeleteUser_WithClients_AndCascadeTrue_DeletesSuccessfully(t *testing.T) {
	user := newTestUser()
	deleteCalled := false
	repo := userRepoReturning(user)
	repo.DeleteFn = func(ctx context.Context, id uuid.UUID) error {
		deleteCalled = true
		return nil
	}
	clientRepo := &mocks.MockClientRepository{
		CountByUserIDFn: func(ctx context.Context, userID uuid.UUID) (int64, error) {
			return 3, nil
		},
	}
	svc := services.NewUserService(repo, clientRepo)

	err := svc.DeleteUser(context.Background(), user.ID, true)

	require.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestDeleteUser_WithClients_AndCascadeFalse_ReturnsError(t *testing.T) {
	user := newTestUser()
	repo := userRepoReturning(user)
	clientRepo := &mocks.MockClientRepository{
		CountByUserIDFn: func(ctx context.Context, userID uuid.UUID) (int64, error) {
			return 2, nil
		},
	}
	svc := services.NewUserService(repo, clientRepo)

	err := svc.DeleteUser(context.Background(), user.ID, false)

	assert.ErrorIs(t, err, services.ErrHasAssociatedClients)
}

func TestDeleteUser_WithNonExistingUser_ReturnsError(t *testing.T) {
	svc := services.NewUserService(userRepoReturning(nil), &mocks.MockClientRepository{})

	err := svc.DeleteUser(context.Background(), uuid.New(), true)

	assert.ErrorIs(t, err, services.ErrUserNotFound)
}

func TestDeleteUser_WhenDeleteFails_ReturnsError(t *testing.T) {
	user := newTestUser()
	repo := userRepoReturning(user)
	repo.DeleteFn = func(ctx context.Context, id uuid.UUID) error {
		return errors.New("db error")
	}
	svc := services.NewUserService(repo, &mocks.MockClientRepository{})

	err := svc.DeleteUser(context.Background(), user.ID, true)

	assert.EqualError(t, err, "db error")
}
