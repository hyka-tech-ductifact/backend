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
// Register
// =============================================================================

func TestRegister_WithValidData_ReturnsUserAndToken(t *testing.T) {
	// ARRANGE
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	mockToken := &mocks.MockTokenProvider{
		GenerateTokenFn: func(userID uuid.UUID, email string) (string, error) {
			return "jwt-token-123", nil
		},
	}

	svc := services.NewAuthService(mockRepo, mockToken)

	// ACT
	user, token, err := svc.Register(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, "Juan", user.Name)
	assert.Equal(t, "juan@example.com", user.Email)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "jwt-token-123", token)
}

func TestRegister_WithDuplicateEmail_ReturnsError(t *testing.T) {
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
	mockToken := &mocks.MockTokenProvider{}

	svc := services.NewAuthService(mockRepo, mockToken)

	// ACT
	user, token, err := svc.Register(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT
	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.ErrorIs(t, err, services.ErrEmailTaken)
}

func TestRegister_WithEmptyName_ReturnsError(t *testing.T) {
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	mockToken := &mocks.MockTokenProvider{}

	svc := services.NewAuthService(mockRepo, mockToken)

	user, token, err := svc.Register(context.Background(), "", "juan@example.com", "securepass123")

	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.ErrorIs(t, err, entities.ErrEmptyUserName)
}

func TestRegister_WithInvalidEmail_ReturnsError(t *testing.T) {
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	mockToken := &mocks.MockTokenProvider{}

	svc := services.NewAuthService(mockRepo, mockToken)

	user, token, err := svc.Register(context.Background(), "Juan", "not-an-email", "securepass123")

	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.Error(t, err)
}

func TestRegister_WithShortPassword_ReturnsError(t *testing.T) {
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	mockToken := &mocks.MockTokenProvider{}

	svc := services.NewAuthService(mockRepo, mockToken)

	user, token, err := svc.Register(context.Background(), "Juan", "juan@example.com", "short")

	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.ErrorIs(t, err, valueobjects.ErrPasswordTooShort)
}

func TestRegister_WhenRepoCreateFails_ReturnsError(t *testing.T) {
	// ARRANGE
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
		CreateFn: func(ctx context.Context, user *entities.User) error {
			return errors.New("db connection lost")
		},
	}
	mockToken := &mocks.MockTokenProvider{}

	svc := services.NewAuthService(mockRepo, mockToken)

	// ACT
	user, token, err := svc.Register(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT
	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.EqualError(t, err, "db connection lost")
}

func TestRegister_WhenTokenGenerationFails_ReturnsError(t *testing.T) {
	// ARRANGE
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	mockToken := &mocks.MockTokenProvider{
		GenerateTokenFn: func(userID uuid.UUID, email string) (string, error) {
			return "", errors.New("token generation failed")
		},
	}

	svc := services.NewAuthService(mockRepo, mockToken)

	// ACT
	user, token, err := svc.Register(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT
	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.EqualError(t, err, "token generation failed")
}

// =============================================================================
// Login
// =============================================================================

func TestLogin_WithValidCredentials_ReturnsUserAndToken(t *testing.T) {
	// ARRANGE: create a real bcrypt hash for "securepass123"
	pwd, _ := valueobjects.NewPassword("securepass123")

	storedUser := &entities.User{
		ID:           uuid.New(),
		Name:         "Juan",
		Email:        "juan@example.com",
		PasswordHash: pwd.Hash(),
	}

	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			if email == "juan@example.com" {
				return storedUser, nil
			}
			return nil, errors.New("not found")
		},
	}
	mockToken := &mocks.MockTokenProvider{
		GenerateTokenFn: func(userID uuid.UUID, email string) (string, error) {
			return "jwt-token-456", nil
		},
	}

	svc := services.NewAuthService(mockRepo, mockToken)

	// ACT
	user, token, err := svc.Login(context.Background(), "juan@example.com", "securepass123")

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, storedUser.ID, user.ID)
	assert.Equal(t, "Juan", user.Name)
	assert.Equal(t, "juan@example.com", user.Email)
	assert.Equal(t, "jwt-token-456", token)
}

func TestLogin_WithWrongPassword_ReturnsInvalidCredentials(t *testing.T) {
	// ARRANGE
	pwd, _ := valueobjects.NewPassword("securepass123")

	storedUser := &entities.User{
		ID:           uuid.New(),
		Name:         "Juan",
		Email:        "juan@example.com",
		PasswordHash: pwd.Hash(),
	}

	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return storedUser, nil
		},
	}
	mockToken := &mocks.MockTokenProvider{}

	svc := services.NewAuthService(mockRepo, mockToken)

	// ACT
	user, token, err := svc.Login(context.Background(), "juan@example.com", "wrongpassword")

	// ASSERT
	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.ErrorIs(t, err, services.ErrInvalidCredentials)
}

func TestLogin_WithNonExistentEmail_ReturnsInvalidCredentials(t *testing.T) {
	// ARRANGE
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, errors.New("not found")
		},
	}
	mockToken := &mocks.MockTokenProvider{}

	svc := services.NewAuthService(mockRepo, mockToken)

	// ACT
	user, token, err := svc.Login(context.Background(), "noexiste@example.com", "securepass123")

	// ASSERT: same generic error — don't reveal if email exists
	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.ErrorIs(t, err, services.ErrInvalidCredentials)
}

func TestLogin_WhenTokenGenerationFails_ReturnsError(t *testing.T) {
	// ARRANGE
	pwd, _ := valueobjects.NewPassword("securepass123")

	storedUser := &entities.User{
		ID:           uuid.New(),
		Name:         "Juan",
		Email:        "juan@example.com",
		PasswordHash: pwd.Hash(),
	}

	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return storedUser, nil
		},
	}
	mockToken := &mocks.MockTokenProvider{
		GenerateTokenFn: func(userID uuid.UUID, email string) (string, error) {
			return "", errors.New("token signing failed")
		},
	}

	svc := services.NewAuthService(mockRepo, mockToken)

	// ACT
	user, token, err := svc.Login(context.Background(), "juan@example.com", "securepass123")

	// ASSERT
	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.EqualError(t, err, "token signing failed")
}

func TestRegister_WhenGetByEmailFails_ReturnsError(t *testing.T) {
	// ARRANGE: GetByEmail returns a non-"not found" error (e.g. DB failure)
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, errors.New("db connection lost")
		},
	}
	mockToken := &mocks.MockTokenProvider{}

	svc := services.NewAuthService(mockRepo, mockToken)

	// ACT
	user, token, err := svc.Register(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT: DB error is propagated instead of silently ignored
	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.EqualError(t, err, "db connection lost")
}
