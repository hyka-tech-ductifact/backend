package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ductifact/internal/application/services"
	"ductifact/internal/domain/entities"
	"ductifact/test/unit/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CreateUser
// =============================================================================

func TestCreateUser_WithValidData_ReturnsUser(t *testing.T) {
	// ARRANGE
	mockRepo := &mocks.MockUserRepository{}

	svc := services.NewUserService(mockRepo)

	// ACT
	user, err := svc.CreateUser(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, "Juan", user.Name)
	assert.Equal(t, "juan@example.com", user.Email)
	assert.NotEmpty(t, user.ID)
}

func TestCreateUser_WithDuplicateEmail_ReturnsError(t *testing.T) {
	// ARRANGE
	existingUser := &entities.User{
		ID:    uuid.New(),
		Name:  "Existing",
		Email: "juan@example.com",
	}

	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return existingUser, nil
		},
	}

	svc := services.NewUserService(mockRepo)

	// ACT
	user, err := svc.CreateUser(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT
	assert.Nil(t, user)
	assert.ErrorIs(t, err, services.ErrEmailAlreadyInUse)
}

func TestCreateUser_WithEmptyName_ReturnsError(t *testing.T) {
	mockRepo := &mocks.MockUserRepository{}
	svc := services.NewUserService(mockRepo)

	user, err := svc.CreateUser(context.Background(), "", "juan@example.com", "securepass123")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, entities.ErrEmptyUserName)
}

func TestCreateUser_WithInvalidEmail_ReturnsError(t *testing.T) {
	mockRepo := &mocks.MockUserRepository{}
	svc := services.NewUserService(mockRepo)

	user, err := svc.CreateUser(context.Background(), "Juan", "not-an-email", "securepass123")

	assert.Nil(t, user)
	assert.Error(t, err)
}

func TestCreateUser_WhenRepoFails_ReturnsError(t *testing.T) {
	// ARRANGE
	mockRepo := &mocks.MockUserRepository{
		CreateFn: func(ctx context.Context, user *entities.User) error {
			return errors.New("db connection lost")
		},
	}

	svc := services.NewUserService(mockRepo)

	// ACT
	user, err := svc.CreateUser(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT
	assert.Nil(t, user)
	assert.EqualError(t, err, "db connection lost")
}

// =============================================================================
// GetUserByID
// =============================================================================

func TestGetUserByID_WithExistingUser_ReturnsUser(t *testing.T) {
	expectedID := uuid.New()
	expectedUser := &entities.User{
		ID:    expectedID,
		Name:  "Juan",
		Email: "juan@example.com",
	}

	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			if id == expectedID {
				return expectedUser, nil
			}
			return nil, errors.New("not found")
		},
	}

	svc := services.NewUserService(mockRepo)

	user, err := svc.GetUserByID(context.Background(), expectedID)

	require.NoError(t, err)
	assert.Equal(t, expectedUser.Name, user.Name)
	assert.Equal(t, expectedUser.Email, user.Email)
	assert.Equal(t, expectedID, user.ID)
}

func TestGetUserByID_WithNonExistingUser_ReturnsError(t *testing.T) {
	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return nil, errors.New("not found")
		},
	}

	svc := services.NewUserService(mockRepo)

	user, err := svc.GetUserByID(context.Background(), uuid.New())

	assert.Nil(t, user)
	assert.ErrorIs(t, err, services.ErrUserNotFound)
}

// =============================================================================
// UpdateUser
// =============================================================================

func TestUpdateUser_WithNewName_UpdatesOnlyName(t *testing.T) {
	existingUser := &entities.User{
		ID:        uuid.New(),
		Name:      "Juan",
		Email:     "juan@example.com",
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			cp := *existingUser
			return &cp, nil
		},
		UpdateFn: func(ctx context.Context, user *entities.User) error {
			return nil
		},
	}

	svc := services.NewUserService(mockRepo)
	newName := "Pedro"

	user, err := svc.UpdateUser(context.Background(), existingUser.ID, &newName, nil)

	require.NoError(t, err)
	assert.Equal(t, "Pedro", user.Name)
	assert.Equal(t, "juan@example.com", user.Email)
}

func TestUpdateUser_WithNewEmail_UpdatesOnlyEmail(t *testing.T) {
	existingUser := &entities.User{
		ID:        uuid.New(),
		Name:      "Juan",
		Email:     "juan@example.com",
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			cp := *existingUser
			return &cp, nil
		},
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, errors.New("not found")
		},
		UpdateFn: func(ctx context.Context, user *entities.User) error {
			return nil
		},
	}

	svc := services.NewUserService(mockRepo)
	newEmail := "pedro@example.com"

	user, err := svc.UpdateUser(context.Background(), existingUser.ID, nil, &newEmail)

	require.NoError(t, err)
	assert.Equal(t, "Juan", user.Name)
	assert.Equal(t, "pedro@example.com", user.Email)
}

func TestUpdateUser_WithDuplicateEmail_ReturnsError(t *testing.T) {
	userID := uuid.New()
	existingUser := &entities.User{
		ID:    userID,
		Name:  "Juan",
		Email: "juan@example.com",
	}

	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			cp := *existingUser
			return &cp, nil
		},
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return &entities.User{ID: uuid.New(), Email: email}, nil
		},
	}

	svc := services.NewUserService(mockRepo)
	newEmail := "taken@example.com"

	user, err := svc.UpdateUser(context.Background(), userID, nil, &newEmail)

	assert.Nil(t, user)
	assert.ErrorIs(t, err, services.ErrEmailAlreadyInUse)
}

func TestUpdateUser_WithSameEmail_DoesNotCheckUniqueness(t *testing.T) {
	userID := uuid.New()
	existingUser := &entities.User{
		ID:        userID,
		Name:      "Juan",
		Email:     "juan@example.com",
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	getByEmailCalled := false
	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			cp := *existingUser
			return &cp, nil
		},
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			getByEmailCalled = true
			return nil, nil
		},
		UpdateFn: func(ctx context.Context, user *entities.User) error {
			return nil
		},
	}

	svc := services.NewUserService(mockRepo)
	sameEmail := "juan@example.com"

	user, err := svc.UpdateUser(context.Background(), userID, nil, &sameEmail)

	require.NoError(t, err)
	assert.Equal(t, "juan@example.com", user.Email)
	assert.False(t, getByEmailCalled, "GetByEmail should not be called when email does not change")
}

func TestUpdateUser_WithNonExistingUser_ReturnsError(t *testing.T) {
	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return nil, errors.New("not found")
		},
	}

	svc := services.NewUserService(mockRepo)
	newName := "Pedro"

	user, err := svc.UpdateUser(context.Background(), uuid.New(), &newName, nil)

	assert.Nil(t, user)
	assert.ErrorIs(t, err, services.ErrUserNotFound)
}

func TestUpdateUser_WhenRepoFails_ReturnsError(t *testing.T) {
	existingUser := &entities.User{
		ID:    uuid.New(),
		Name:  "Juan",
		Email: "juan@example.com",
	}

	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			cp := *existingUser
			return &cp, nil
		},
		UpdateFn: func(ctx context.Context, user *entities.User) error {
			return errors.New("db write failed")
		},
	}

	svc := services.NewUserService(mockRepo)
	newName := "Pedro"

	user, err := svc.UpdateUser(context.Background(), existingUser.ID, &newName, nil)

	assert.Nil(t, user)
	assert.EqualError(t, err, "db write failed")
}

func TestUpdateUser_UpdatesTimestamp(t *testing.T) {
	oldTime := time.Now().Add(-time.Hour)
	existingUser := &entities.User{
		ID:        uuid.New(),
		Name:      "Juan",
		Email:     "juan@example.com",
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
	}

	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			cp := *existingUser
			return &cp, nil
		},
		UpdateFn: func(ctx context.Context, user *entities.User) error {
			return nil
		},
	}

	svc := services.NewUserService(mockRepo)
	newName := "Pedro"

	user, err := svc.UpdateUser(context.Background(), existingUser.ID, &newName, nil)

	require.NoError(t, err)
	assert.True(t, user.UpdatedAt.After(oldTime), "UpdatedAt must be newer than the original")
}
