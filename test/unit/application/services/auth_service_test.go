package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ductifact/internal/application/ports"
	"ductifact/internal/application/services"
	"ductifact/internal/application/usecases"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"
	"ductifact/internal/domain/valueobjects"
	"ductifact/test/unit/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestAuthService creates an AuthService with a no-op blacklist and test durations.
func newTestAuthService(repo *mocks.MockUserRepository, token *mocks.MockTokenProvider) usecases.AuthService {
	return services.NewAuthService(repo, token, &mocks.MockTokenBlacklist{}, 15*time.Minute, 7*24*time.Hour)
}

// newTestAuthServiceWithBlacklist creates an AuthService with a custom blacklist.
func newTestAuthServiceWithBlacklist(
	repo *mocks.MockUserRepository,
	token *mocks.MockTokenProvider,
	blacklist *mocks.MockTokenBlacklist,
) usecases.AuthService {
	return services.NewAuthService(repo, token, blacklist, 15*time.Minute, 7*24*time.Hour)
}

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
		GenerateTokenPairFn: func(userID uuid.UUID, email string) (*ports.TokenPair, error) {
			return &ports.TokenPair{
				AccessToken:  "access-token-123",
				RefreshToken: "refresh-token-123",
			}, nil
		},
	}

	svc := newTestAuthService(mockRepo, mockToken)

	// ACT
	user, tokens, err := svc.Register(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, "Juan", user.Name)
	assert.Equal(t, "juan@example.com", user.Email)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "access-token-123", tokens.AccessToken)
	assert.Equal(t, "refresh-token-123", tokens.RefreshToken)
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

	svc := newTestAuthService(mockRepo, mockToken)

	// ACT
	user, tokens, err := svc.Register(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT
	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.ErrorIs(t, err, services.ErrEmailAlreadyInUse)
}

func TestRegister_WithEmptyName_ReturnsError(t *testing.T) {
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	mockToken := &mocks.MockTokenProvider{}

	svc := newTestAuthService(mockRepo, mockToken)

	user, tokens, err := svc.Register(context.Background(), "", "juan@example.com", "securepass123")

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.ErrorIs(t, err, entities.ErrEmptyUserName)
}

func TestRegister_WithInvalidEmail_ReturnsError(t *testing.T) {
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	mockToken := &mocks.MockTokenProvider{}

	svc := newTestAuthService(mockRepo, mockToken)

	user, tokens, err := svc.Register(context.Background(), "Juan", "not-an-email", "securepass123")

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.Error(t, err)
}

func TestRegister_WithShortPassword_ReturnsError(t *testing.T) {
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	mockToken := &mocks.MockTokenProvider{}

	svc := newTestAuthService(mockRepo, mockToken)

	user, tokens, err := svc.Register(context.Background(), "Juan", "juan@example.com", "short")

	assert.Nil(t, user)
	assert.Nil(t, tokens)
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

	svc := newTestAuthService(mockRepo, mockToken)

	// ACT
	user, tokens, err := svc.Register(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT
	assert.Nil(t, user)
	assert.Nil(t, tokens)
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
		GenerateTokenPairFn: func(userID uuid.UUID, email string) (*ports.TokenPair, error) {
			return nil, errors.New("token generation failed")
		},
	}

	svc := newTestAuthService(mockRepo, mockToken)

	// ACT
	user, tokens, err := svc.Register(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT
	assert.Nil(t, user)
	assert.Nil(t, tokens)
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
		GenerateTokenPairFn: func(userID uuid.UUID, email string) (*ports.TokenPair, error) {
			return &ports.TokenPair{
				AccessToken:  "access-token-456",
				RefreshToken: "refresh-token-456",
			}, nil
		},
	}

	svc := newTestAuthService(mockRepo, mockToken)

	// ACT
	user, tokens, err := svc.Login(context.Background(), "juan@example.com", "securepass123")

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, storedUser.ID, user.ID)
	assert.Equal(t, "Juan", user.Name)
	assert.Equal(t, "juan@example.com", user.Email)
	assert.Equal(t, "access-token-456", tokens.AccessToken)
	assert.Equal(t, "refresh-token-456", tokens.RefreshToken)
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

	svc := newTestAuthService(mockRepo, mockToken)

	// ACT
	user, tokens, err := svc.Login(context.Background(), "juan@example.com", "wrongpassword")

	// ASSERT
	assert.Nil(t, user)
	assert.Nil(t, tokens)
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

	svc := newTestAuthService(mockRepo, mockToken)

	// ACT: same generic error — don't reveal if email exists
	user, tokens, err := svc.Login(context.Background(), "noexiste@example.com", "securepass123")

	// ASSERT: same generic error — don't reveal if email exists
	assert.Nil(t, user)
	assert.Nil(t, tokens)
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
		GenerateTokenPairFn: func(userID uuid.UUID, email string) (*ports.TokenPair, error) {
			return nil, errors.New("token signing failed")
		},
	}

	svc := newTestAuthService(mockRepo, mockToken)

	// ACT
	user, tokens, err := svc.Login(context.Background(), "juan@example.com", "securepass123")

	// ASSERT
	assert.Nil(t, user)
	assert.Nil(t, tokens)
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

	svc := newTestAuthService(mockRepo, mockToken)

	// ACT
	user, tokens, err := svc.Register(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT: DB error is propagated instead of silently ignored
	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.EqualError(t, err, "db connection lost")
}

// =============================================================================
// RefreshToken
// =============================================================================

func TestRefreshToken_WithValidRefreshToken_ReturnsNewTokenPair(t *testing.T) {
	// ARRANGE
	userID := uuid.New()

	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return &entities.User{
				ID:    userID,
				Name:  "Juan",
				Email: "juan@example.com",
			}, nil
		},
	}
	mockToken := &mocks.MockTokenProvider{
		ValidateRefreshTokenFn: func(tokenString string) (*ports.TokenClaims, error) {
			return &ports.TokenClaims{
				UserID: userID,
				Email:  "juan@example.com",
			}, nil
		},
		GenerateTokenPairFn: func(uid uuid.UUID, email string) (*ports.TokenPair, error) {
			return &ports.TokenPair{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
			}, nil
		},
	}

	svc := newTestAuthService(mockRepo, mockToken)

	// ACT
	tokens, err := svc.RefreshToken(context.Background(), "old-refresh-token")

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", tokens.AccessToken)
	assert.Equal(t, "new-refresh-token", tokens.RefreshToken)
}

func TestRefreshToken_WithInvalidRefreshToken_ReturnsError(t *testing.T) {
	// ARRANGE
	mockRepo := &mocks.MockUserRepository{}
	mockToken := &mocks.MockTokenProvider{
		ValidateRefreshTokenFn: func(tokenString string) (*ports.TokenClaims, error) {
			return nil, errors.New("invalid token")
		},
	}

	svc := newTestAuthService(mockRepo, mockToken)

	// ACT
	tokens, err := svc.RefreshToken(context.Background(), "garbage-token")

	// ASSERT
	assert.Nil(t, tokens)
	assert.ErrorIs(t, err, services.ErrInvalidRefreshToken)
}

func TestRefreshToken_WhenUserNoLongerExists_ReturnsError(t *testing.T) {
	// ARRANGE
	userID := uuid.New()

	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return nil, errors.New("not found")
		},
	}
	mockToken := &mocks.MockTokenProvider{
		ValidateRefreshTokenFn: func(tokenString string) (*ports.TokenClaims, error) {
			return &ports.TokenClaims{
				UserID: userID,
				Email:  "deleted@example.com",
			}, nil
		},
	}

	svc := newTestAuthService(mockRepo, mockToken)

	// ACT
	tokens, err := svc.RefreshToken(context.Background(), "valid-refresh-token")

	// ASSERT
	assert.Nil(t, tokens)
	assert.ErrorIs(t, err, services.ErrInvalidRefreshToken)
}

func TestRefreshToken_WhenTokenGenerationFails_ReturnsError(t *testing.T) {
	// ARRANGE
	userID := uuid.New()

	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return &entities.User{
				ID:    userID,
				Name:  "Juan",
				Email: "juan@example.com",
			}, nil
		},
	}
	mockToken := &mocks.MockTokenProvider{
		ValidateRefreshTokenFn: func(tokenString string) (*ports.TokenClaims, error) {
			return &ports.TokenClaims{
				UserID: userID,
				Email:  "juan@example.com",
			}, nil
		},
		GenerateTokenPairFn: func(uid uuid.UUID, email string) (*ports.TokenPair, error) {
			return nil, errors.New("signing failure")
		},
	}

	svc := newTestAuthService(mockRepo, mockToken)

	// ACT
	tokens, err := svc.RefreshToken(context.Background(), "valid-refresh-token")

	// ASSERT
	assert.Nil(t, tokens)
	assert.EqualError(t, err, "signing failure")
}

func TestRefreshToken_WithBlacklistedToken_ReturnsError(t *testing.T) {
	// ARRANGE
	blacklist := &mocks.MockTokenBlacklist{
		IsBlacklistedFn: func(token string) bool {
			return token == "revoked-refresh-token"
		},
	}
	mockRepo := &mocks.MockUserRepository{}
	mockToken := &mocks.MockTokenProvider{}

	svc := newTestAuthServiceWithBlacklist(mockRepo, mockToken, blacklist)

	// ACT
	tokens, err := svc.RefreshToken(context.Background(), "revoked-refresh-token")

	// ASSERT
	assert.Nil(t, tokens)
	assert.ErrorIs(t, err, services.ErrInvalidRefreshToken)
}

// =============================================================================
// Logout
// =============================================================================

func TestLogout_BlacklistsBothTokens(t *testing.T) {
	// ARRANGE
	var addedTokens []string
	blacklist := &mocks.MockTokenBlacklist{
		AddFn: func(token string, expiry time.Duration) {
			addedTokens = append(addedTokens, token)
		},
	}
	mockRepo := &mocks.MockUserRepository{}
	mockToken := &mocks.MockTokenProvider{}

	svc := newTestAuthServiceWithBlacklist(mockRepo, mockToken, blacklist)

	// ACT
	err := svc.Logout(context.Background(), "access-token-123", "refresh-token-456")

	// ASSERT
	require.NoError(t, err)
	assert.Contains(t, addedTokens, "access-token-123")
	assert.Contains(t, addedTokens, "refresh-token-456")
	assert.Len(t, addedTokens, 2)
}

func TestLogout_UsesCorrectDurations(t *testing.T) {
	// ARRANGE
	durations := make(map[string]time.Duration)
	blacklist := &mocks.MockTokenBlacklist{
		AddFn: func(token string, expiry time.Duration) {
			durations[token] = expiry
		},
	}
	mockRepo := &mocks.MockUserRepository{}
	mockToken := &mocks.MockTokenProvider{}

	svc := newTestAuthServiceWithBlacklist(mockRepo, mockToken, blacklist)

	// ACT
	_ = svc.Logout(context.Background(), "access-tok", "refresh-tok")

	// ASSERT: access token uses 15min, refresh uses 7 days
	assert.Equal(t, 15*time.Minute, durations["access-tok"])
	assert.Equal(t, 7*24*time.Hour, durations["refresh-tok"])
}
