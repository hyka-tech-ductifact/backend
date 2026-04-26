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

// newTestAuthService creates an AuthService with a no-op blacklist, no-op throttler and test durations.
func newTestAuthService(repo *mocks.MockUserRepository, token *mocks.MockTokenProvider) usecases.AuthService {
	return services.NewAuthService(repo, token, &mocks.MockTokenBlacklist{}, &mocks.MockLoginThrottler{}, &mocks.MockEmailSender{}, 15*time.Minute, 7*24*time.Hour)
}

// newTestAuthServiceWithEmail creates an AuthService with a custom email sender.
func newTestAuthServiceWithEmail(
	repo *mocks.MockUserRepository,
	token *mocks.MockTokenProvider,
	emailSender *mocks.MockEmailSender,
) usecases.AuthService {
	return services.NewAuthService(repo, token, &mocks.MockTokenBlacklist{}, &mocks.MockLoginThrottler{}, emailSender, 15*time.Minute, 7*24*time.Hour)
}

// newTestAuthServiceWithBlacklist creates an AuthService with a custom blacklist.
func newTestAuthServiceWithBlacklist(
	repo *mocks.MockUserRepository,
	token *mocks.MockTokenProvider,
	blacklist *mocks.MockTokenBlacklist,
) usecases.AuthService {
	return services.NewAuthService(repo, token, blacklist, &mocks.MockLoginThrottler{}, &mocks.MockEmailSender{}, 15*time.Minute, 7*24*time.Hour)
}

// newTestAuthServiceWithThrottler creates an AuthService with a custom login throttler.
func newTestAuthServiceWithThrottler(
	repo *mocks.MockUserRepository,
	token *mocks.MockTokenProvider,
	throttler *mocks.MockLoginThrottler,
) usecases.AuthService {
	return services.NewAuthService(repo, token, &mocks.MockTokenBlacklist{}, throttler, &mocks.MockEmailSender{}, 15*time.Minute, 7*24*time.Hour)
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

func TestRegister_SendsWelcomeEmail(t *testing.T) {
	// ARRANGE
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	mockToken := &mocks.MockTokenProvider{
		GenerateTokenPairFn: func(userID uuid.UUID, email string) (*ports.TokenPair, error) {
			return &ports.TokenPair{AccessToken: "a", RefreshToken: "r"}, nil
		},
	}
	mockEmail := &mocks.MockEmailSender{}

	svc := newTestAuthServiceWithEmail(mockRepo, mockToken, mockEmail)

	// ACT
	user, _, err := svc.Register(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT
	require.NoError(t, err)
	require.Len(t, mockEmail.Sent, 1)
	assert.Equal(t, user.Email, mockEmail.Sent[0].To)
	assert.Equal(t, "Welcome to Ductifact", mockEmail.Sent[0].Subject)
	assert.Contains(t, mockEmail.Sent[0].HTML, "Juan")
	assert.Contains(t, mockEmail.Sent[0].Text, "Juan")
}

func TestRegister_SucceedsEvenIfEmailFails(t *testing.T) {
	// ARRANGE
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	mockToken := &mocks.MockTokenProvider{
		GenerateTokenPairFn: func(userID uuid.UUID, email string) (*ports.TokenPair, error) {
			return &ports.TokenPair{AccessToken: "a", RefreshToken: "r"}, nil
		},
	}
	mockEmail := &mocks.MockEmailSender{
		SendFn: func(ctx context.Context, email ports.Email) error {
			return errors.New("smtp down")
		},
	}

	svc := newTestAuthServiceWithEmail(mockRepo, mockToken, mockEmail)

	// ACT
	user, tokens, err := svc.Register(context.Background(), "Juan", "juan@example.com", "securepass123")

	// ASSERT: registration succeeds despite email failure
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotNil(t, tokens)
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

// =============================================================================
// Login — Brute-force protection
// =============================================================================

func TestLogin_WhenAccountIsBlocked_ReturnsAccountLocked(t *testing.T) {
	throttler := &mocks.MockLoginThrottler{
		IsBlockedFn: func(key string) bool { return true },
	}
	mockRepo := &mocks.MockUserRepository{}
	mockToken := &mocks.MockTokenProvider{}

	svc := newTestAuthServiceWithThrottler(mockRepo, mockToken, throttler)

	user, tokens, err := svc.Login(context.Background(), "juan@example.com", "any-password")

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.ErrorIs(t, err, services.ErrAccountLocked)
}

func TestLogin_WithWrongPassword_RecordsFailure(t *testing.T) {
	failureRecorded := false
	throttler := &mocks.MockLoginThrottler{
		RecordFailureFn: func(key string) {
			assert.Equal(t, "juan@example.com", key)
			failureRecorded = true
		},
	}

	existingUser, _ := entities.NewUser("Juan", "juan@example.com", "securepass123")
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return existingUser, nil
		},
	}
	mockToken := &mocks.MockTokenProvider{}

	svc := newTestAuthServiceWithThrottler(mockRepo, mockToken, throttler)

	_, _, err := svc.Login(context.Background(), "juan@example.com", "wrong-password")

	assert.ErrorIs(t, err, services.ErrInvalidCredentials)
	assert.True(t, failureRecorded)
}

func TestLogin_WithNonexistentEmail_RecordsFailure(t *testing.T) {
	failureRecorded := false
	throttler := &mocks.MockLoginThrottler{
		RecordFailureFn: func(key string) {
			assert.Equal(t, "unknown@example.com", key)
			failureRecorded = true
		},
	}

	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	mockToken := &mocks.MockTokenProvider{}

	svc := newTestAuthServiceWithThrottler(mockRepo, mockToken, throttler)

	_, _, err := svc.Login(context.Background(), "unknown@example.com", "any-password")

	assert.ErrorIs(t, err, services.ErrInvalidCredentials)
	assert.True(t, failureRecorded)
}

func TestLogin_WithCorrectPassword_ResetsThrottler(t *testing.T) {
	resetCalled := false
	throttler := &mocks.MockLoginThrottler{
		ResetFn: func(key string) {
			assert.Equal(t, "juan@example.com", key)
			resetCalled = true
		},
	}

	existingUser, _ := entities.NewUser("Juan", "juan@example.com", "securepass123")
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return existingUser, nil
		},
	}
	mockToken := &mocks.MockTokenProvider{
		GenerateTokenPairFn: func(userID uuid.UUID, email string) (*ports.TokenPair, error) {
			return &ports.TokenPair{AccessToken: "at", RefreshToken: "rt"}, nil
		},
	}

	svc := newTestAuthServiceWithThrottler(mockRepo, mockToken, throttler)

	_, _, err := svc.Login(context.Background(), "juan@example.com", "securepass123")

	require.NoError(t, err)
	assert.True(t, resetCalled)
}

func TestLogin_WhenBlocked_DoesNotQueryDatabase(t *testing.T) {
	dbQueried := false
	throttler := &mocks.MockLoginThrottler{
		IsBlockedFn: func(key string) bool { return true },
	}

	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			dbQueried = true
			return nil, repositories.ErrNotFound
		},
	}
	mockToken := &mocks.MockTokenProvider{}

	svc := newTestAuthServiceWithThrottler(mockRepo, mockToken, throttler)

	_, _, err := svc.Login(context.Background(), "juan@example.com", "any-password")
	require.Error(t, err)

	assert.False(t, dbQueried, "should not query DB when account is blocked")
}
