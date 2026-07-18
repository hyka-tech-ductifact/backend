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
	return services.NewAuthService(
		repo,
		&mocks.MockOneTimeOTPRepository{},
		token,
		&mocks.MockTokenBlacklist{},
		&mocks.MockLoginThrottler{},
		&mocks.MockEmailSender{},
		&mocks.MockRateLimiter{},
		15*time.Minute,
		7*24*time.Hour,
		24*time.Hour,
		1*time.Hour,
	)
}

// newTestAuthServiceWithBlacklist creates an AuthService with a custom blacklist.
func newTestAuthServiceWithBlacklist(
	repo *mocks.MockUserRepository,
	token *mocks.MockTokenProvider,
	blacklist *mocks.MockTokenBlacklist,
) usecases.AuthService {
	return services.NewAuthService(
		repo,
		&mocks.MockOneTimeOTPRepository{},
		token,
		blacklist,
		&mocks.MockLoginThrottler{},
		&mocks.MockEmailSender{},
		&mocks.MockRateLimiter{},
		15*time.Minute,
		7*24*time.Hour,
		24*time.Hour,
		1*time.Hour,
	)
}

// newTestAuthServiceWithThrottler creates an AuthService with a custom login throttler.
func newTestAuthServiceWithThrottler(
	repo *mocks.MockUserRepository,
	token *mocks.MockTokenProvider,
	throttler *mocks.MockLoginThrottler,
) usecases.AuthService {
	return services.NewAuthService(
		repo,
		&mocks.MockOneTimeOTPRepository{},
		token,
		&mocks.MockTokenBlacklist{},
		throttler,
		&mocks.MockEmailSender{},
		&mocks.MockRateLimiter{},
		15*time.Minute,
		7*24*time.Hour,
		24*time.Hour,
		1*time.Hour,
	)
}

// newTestAuthServiceForRegistration builds an AuthService wired for the
// email-first OTP registration flow.
func newTestAuthServiceForRegistration(
	userRepo *mocks.MockUserRepository,
	otpRepo *mocks.MockOneTimeOTPRepository,
	token *mocks.MockTokenProvider,
	email *mocks.MockEmailSender,
) usecases.AuthService {
	return newTestAuthServiceForRegistrationWithNoticeLimiter(
		userRepo,
		otpRepo,
		token,
		email,
		&mocks.MockRateLimiter{},
	)
}

func newTestAuthServiceForRegistrationWithNoticeLimiter(
	userRepo *mocks.MockUserRepository,
	otpRepo *mocks.MockOneTimeOTPRepository,
	token *mocks.MockTokenProvider,
	email *mocks.MockEmailSender,
	registrationNoticeLimiter *mocks.MockRateLimiter,
) usecases.AuthService {
	return services.NewAuthService(
		userRepo,
		otpRepo,
		token,
		&mocks.MockTokenBlacklist{},
		&mocks.MockLoginThrottler{},
		email,
		registrationNoticeLimiter,
		15*time.Minute,
		7*24*time.Hour,
		15*time.Minute,
		1*time.Hour,
	)
}

// =============================================================================
// StartRegistration
// =============================================================================

func TestStartRegistration_NewEmail_SavesOTPAndSendsEmail(t *testing.T) {
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	otpRepo := &mocks.MockOneTimeOTPRepository{}
	email := &mocks.MockEmailSender{}
	svc := newTestAuthServiceForRegistration(userRepo, otpRepo, &mocks.MockTokenProvider{}, email)

	err := svc.StartRegistration(context.Background(), "juan@example.com", "")

	require.NoError(t, err)
	require.Len(t, otpRepo.Saved, 1)
	assert.Equal(t, "juan@example.com", otpRepo.Saved[0].Email)
	require.Len(t, email.Sent, 1)
	assert.Equal(t, "juan@example.com", email.Sent[0].To)
}

func TestStartRegistration_ExistingUser_SendsLocalizedNoticeWithoutOTPOrLink(t *testing.T) {
	existing, _ := entities.NewUser(entities.CreateUserParams{
		Name: "Juan", Email: "juan@example.com", Password: "securepass123", Locale: "es",
	})
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return existing, nil
		},
	}
	otpRepo := &mocks.MockOneTimeOTPRepository{}
	email := &mocks.MockEmailSender{}
	var limitedKey string
	limiter := &mocks.MockRateLimiter{
		AllowFn: func(key string) bool {
			limitedKey = key
			return true
		},
	}
	svc := newTestAuthServiceForRegistrationWithNoticeLimiter(
		userRepo, otpRepo, &mocks.MockTokenProvider{}, email, limiter,
	)

	err := svc.StartRegistration(context.Background(), "juan@example.com", "en")

	require.NoError(t, err)
	assert.Empty(t, otpRepo.Saved, "no OTP should be created for an existing account")
	require.Len(t, email.Sent, 1)
	assert.Equal(t, "registration-account-exists:juan@example.com", limitedKey)
	assert.Equal(t, "juan@example.com", email.Sent[0].To)
	assert.Equal(t, "Ya tienes una cuenta en Ductifact", email.Sent[0].Subject)
	assert.Contains(t, email.Sent[0].Text, "plataforma que prefieras")
	assert.NotContains(t, email.Sent[0].HTML, "href=")
	assert.NotContains(t, email.Sent[0].Text, "http")
}

func TestStartRegistration_ExistingUser_RecentNoticeDoesNotSendAnother(t *testing.T) {
	existing, _ := entities.NewUser(entities.CreateUserParams{
		Name: "Juan", Email: "juan@example.com", Password: "securepass123", Locale: "es",
	})
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return existing, nil
		},
	}
	email := &mocks.MockEmailSender{}
	limiter := &mocks.MockRateLimiter{
		AllowFn: func(key string) bool { return false },
	}
	svc := newTestAuthServiceForRegistrationWithNoticeLimiter(
		userRepo, &mocks.MockOneTimeOTPRepository{}, &mocks.MockTokenProvider{}, email, limiter,
	)

	err := svc.StartRegistration(context.Background(), "juan@example.com", "es")

	require.NoError(t, err)
	assert.Empty(t, email.Sent, "a recent notice must suppress another email")
}

func TestStartRegistration_PendingOTP_ReturnsErrOTPAlreadyPending(t *testing.T) {
	pendingOTP, _, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return pendingOTP, nil
		},
	}
	emailSender := &mocks.MockEmailSender{}
	svc := newTestAuthServiceForRegistration(userRepo, otpRepo, &mocks.MockTokenProvider{}, emailSender)

	err := svc.StartRegistration(context.Background(), "juan@example.com", "")

	assert.ErrorIs(t, err, services.ErrOTPAlreadyPending)
	assert.Empty(t, otpRepo.Created, "no new OTP should be created")
	assert.Empty(t, emailSender.Sent, "no email should be sent")
}

func TestStartRegistration_ExpiredOTP_GeneratesNewOne(t *testing.T) {
	expiredOTP, _, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, -1*time.Minute)
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return expiredOTP, nil
		},
	}
	emailSender := &mocks.MockEmailSender{}
	svc := newTestAuthServiceForRegistration(userRepo, otpRepo, &mocks.MockTokenProvider{}, emailSender)

	err := svc.StartRegistration(context.Background(), "juan@example.com", "")

	require.NoError(t, err)
	require.Len(t, otpRepo.Created, 1, "a new OTP should be created after expiry")
	require.Len(t, emailSender.Sent, 1, "a new email should be sent")
}

func TestStartRegistration_InvalidEmail_ReturnsError(t *testing.T) {
	svc := newTestAuthServiceForRegistration(
		&mocks.MockUserRepository{}, &mocks.MockOneTimeOTPRepository{},
		&mocks.MockTokenProvider{}, &mocks.MockEmailSender{},
	)

	err := svc.StartRegistration(context.Background(), "not-an-email", "")

	assert.Error(t, err)
}

// =============================================================================
// Register (complete)
// =============================================================================

func TestRegister_WithValidCode_CreatesUserAndReturnsToken(t *testing.T) {
	otp, code, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return otp, nil
		},
	}
	token := &mocks.MockTokenProvider{
		GenerateTokenPairFn: func(userID uuid.UUID, email string) (*ports.TokenPair, error) {
			return &ports.TokenPair{AccessToken: "a", RefreshToken: "r"}, nil
		},
	}
	svc := newTestAuthServiceForRegistration(userRepo, otpRepo, token, &mocks.MockEmailSender{})

	user, tokens, err := svc.Register(context.Background(), "juan@example.com", code, "Juan", "securepass123", "")

	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, tokens)
	assert.Equal(t, "Juan", user.Name)
	assert.Equal(t, "juan@example.com", user.Email)
	assert.Equal(t, "a", tokens.AccessToken)
}

func TestRegister_WithNoOTP_ReturnsInvalidOTP(t *testing.T) {
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return nil, repositories.ErrNotFound
		},
	}
	svc := newTestAuthServiceForRegistration(
		&mocks.MockUserRepository{}, otpRepo, &mocks.MockTokenProvider{}, &mocks.MockEmailSender{},
	)

	user, tokens, err := svc.Register(context.Background(), "juan@example.com", "123456", "Juan", "securepass123", "")

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.ErrorIs(t, err, entities.ErrInvalidOTP)
}

func TestRegister_WithWrongCode_IncrementsAttemptsAndReturnsError(t *testing.T) {
	otp, _, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	incremented := false
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return otp, nil
		},
		IncrementAttemptsFn: func(ctx context.Context, id uuid.UUID) error {
			incremented = true
			return nil
		},
	}
	svc := newTestAuthServiceForRegistration(
		&mocks.MockUserRepository{}, otpRepo, &mocks.MockTokenProvider{}, &mocks.MockEmailSender{},
	)

	user, tokens, err := svc.Register(context.Background(), "juan@example.com", "000000", "Juan", "securepass123", "")

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.ErrorIs(t, err, entities.ErrInvalidOTP)
	assert.True(t, incremented, "a failed attempt should be recorded")
}

func TestRegister_WithExpiredOTP_ReturnsInvalidOTP(t *testing.T) {
	otp, code, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	otp.ExpiresAt = time.Now().Add(-time.Minute) // force expiry
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return otp, nil
		},
	}
	svc := newTestAuthServiceForRegistration(
		&mocks.MockUserRepository{}, otpRepo, &mocks.MockTokenProvider{}, &mocks.MockEmailSender{},
	)

	user, tokens, err := svc.Register(context.Background(), "juan@example.com", code, "Juan", "securepass123", "")

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.ErrorIs(t, err, entities.ErrInvalidOTP)
}

func TestRegister_WithDuplicateEmail_ReturnsEmailInUse(t *testing.T) {
	otp, code, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	existing, _ := entities.NewUser(entities.CreateUserParams{
		Name: "Existing", Email: "juan@example.com", Password: "securepass123", Locale: "en",
	})
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return existing, nil
		},
	}
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return otp, nil
		},
	}
	svc := newTestAuthServiceForRegistration(userRepo, otpRepo, &mocks.MockTokenProvider{}, &mocks.MockEmailSender{})

	user, tokens, err := svc.Register(context.Background(), "juan@example.com", code, "Juan", "securepass123", "")

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.ErrorIs(t, err, services.ErrEmailAlreadyInUse)
}

func TestRegister_WithShortPassword_ReturnsError(t *testing.T) {
	otp, code, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return otp, nil
		},
	}
	svc := newTestAuthServiceForRegistration(userRepo, otpRepo, &mocks.MockTokenProvider{}, &mocks.MockEmailSender{})

	user, tokens, err := svc.Register(context.Background(), "juan@example.com", code, "Juan", "short", "")

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.ErrorIs(t, err, valueobjects.ErrPasswordTooShort)
}

func TestRegister_WithEmptyName_ReturnsError(t *testing.T) {
	otp, code, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return otp, nil
		},
	}
	svc := newTestAuthServiceForRegistration(userRepo, otpRepo, &mocks.MockTokenProvider{}, &mocks.MockEmailSender{})

	user, tokens, err := svc.Register(context.Background(), "juan@example.com", code, "", "securepass123", "")

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.ErrorIs(t, err, entities.ErrEmptyUserName)
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
	otp, code, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, errors.New("db connection lost")
		},
	}
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return otp, nil
		},
	}
	svc := newTestAuthServiceForRegistration(mockRepo, otpRepo, &mocks.MockTokenProvider{}, &mocks.MockEmailSender{})

	// ACT
	user, tokens, err := svc.Register(context.Background(), "juan@example.com", code, "Juan", "securepass123", "")

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

	existingUser, _ := entities.NewUser(entities.CreateUserParams{
		Name: "Juan", Email: "juan@example.com", Password: "securepass123", Locale: "en",
	})
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

	existingUser, _ := entities.NewUser(entities.CreateUserParams{
		Name: "Juan", Email: "juan@example.com", Password: "securepass123", Locale: "en",
	})
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

// =============================================================================
// ChangePassword
// =============================================================================

func TestChangePassword_WithValidData_UpdatesPassword(t *testing.T) {
	// ARRANGE — create a user with a known password hash
	pwd, _ := valueobjects.NewPassword("oldpass123")
	user := &entities.User{
		ID:           uuid.New(),
		Name:         "Juan",
		Email:        "juan@example.com",
		PasswordHash: pwd.Hash(),
		Locale:       "en",
	}
	var updatedUser *entities.User
	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return user, nil
		},
		UpdateFn: func(ctx context.Context, u *entities.User) error {
			updatedUser = u
			return nil
		},
	}
	svc := newTestAuthService(mockRepo, &mocks.MockTokenProvider{})

	// ACT
	err := svc.ChangePassword(context.Background(), user.ID, "oldpass123", "newpass456")

	// ASSERT
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	// Verify the new password works
	newPwd := valueobjects.NewPasswordFromHash(updatedUser.PasswordHash)
	assert.NoError(t, newPwd.Compare("newpass456"))
	// Verify the old password no longer works
	assert.Error(t, newPwd.Compare("oldpass123"))
}

func TestChangePassword_WithWrongCurrentPassword_ReturnsError(t *testing.T) {
	pwd, _ := valueobjects.NewPassword("oldpass123")
	user := &entities.User{
		ID:           uuid.New(),
		Name:         "Juan",
		Email:        "juan@example.com",
		PasswordHash: pwd.Hash(),
		Locale:       "en",
	}
	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return user, nil
		},
	}
	svc := newTestAuthService(mockRepo, &mocks.MockTokenProvider{})

	err := svc.ChangePassword(context.Background(), user.ID, "wrongpassword", "newpass456")

	assert.ErrorIs(t, err, services.ErrInvalidCurrentPassword)
}

func TestChangePassword_WithInvalidNewPassword_ReturnsError(t *testing.T) {
	pwd, _ := valueobjects.NewPassword("oldpass123")
	user := &entities.User{
		ID:           uuid.New(),
		Name:         "Juan",
		Email:        "juan@example.com",
		PasswordHash: pwd.Hash(),
		Locale:       "en",
	}
	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return user, nil
		},
	}
	svc := newTestAuthService(mockRepo, &mocks.MockTokenProvider{})

	// New password too short (less than 8 chars)
	err := svc.ChangePassword(context.Background(), user.ID, "oldpass123", "short")

	assert.ErrorIs(t, err, valueobjects.ErrPasswordTooShort)
}

func TestChangePassword_WithNonExistentUser_ReturnsError(t *testing.T) {
	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	svc := newTestAuthService(mockRepo, &mocks.MockTokenProvider{})

	err := svc.ChangePassword(context.Background(), uuid.New(), "oldpass123", "newpass456")

	assert.ErrorIs(t, err, services.ErrUserNotFound)
}

// =============================================================================
// ForgotPassword
// =============================================================================

func TestForgotPassword_WithExistingEmail_SendsResetOTP(t *testing.T) {
	user := &entities.User{
		ID:     uuid.New(),
		Name:   "Juan",
		Email:  "juan@example.com",
		Locale: "en",
	}
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return user, nil
		},
	}
	otpRepo := &mocks.MockOneTimeOTPRepository{}
	emailSender := &mocks.MockEmailSender{}

	svc := newTestAuthServiceForRegistration(userRepo, otpRepo, &mocks.MockTokenProvider{}, emailSender)

	err := svc.ForgotPassword(context.Background(), "juan@example.com")

	require.NoError(t, err)
	// Verify a password-reset OTP was created
	require.Len(t, otpRepo.Saved, 1)
	assert.Equal(t, entities.OTPPurposePasswordReset, otpRepo.Saved[0].Purpose)
	assert.Equal(t, "juan@example.com", otpRepo.Saved[0].Email)
	// Verify an email was sent
	require.Len(t, emailSender.Sent, 1)
	assert.Equal(t, "juan@example.com", emailSender.Sent[0].To)
}

func TestForgotPassword_WithPendingOTP_DoesNotCreateOrSendAgain(t *testing.T) {
	user := &entities.User{
		ID:     uuid.New(),
		Name:   "Juan",
		Email:  "juan@example.com",
		Locale: "en",
	}
	pendingOTP, _, err := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposePasswordReset, time.Hour)
	require.NoError(t, err)

	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return user, nil
		},
	}
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return pendingOTP, nil
		},
	}
	emailSender := &mocks.MockEmailSender{}

	svc := newTestAuthServiceForRegistration(userRepo, otpRepo, &mocks.MockTokenProvider{}, emailSender)

	err = svc.ForgotPassword(context.Background(), "juan@example.com")

	require.NoError(t, err)
	assert.Empty(t, otpRepo.Saved, "no OTP should be created while one is pending")
	assert.Empty(t, emailSender.Sent, "no email should be sent while one OTP is pending")
}

func TestForgotPassword_WithNonExistingEmail_ReturnsNilSilently(t *testing.T) {
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return nil, repositories.ErrNotFound
		},
	}
	otpRepo := &mocks.MockOneTimeOTPRepository{}
	emailSender := &mocks.MockEmailSender{}

	svc := newTestAuthServiceForRegistration(userRepo, otpRepo, &mocks.MockTokenProvider{}, emailSender)

	err := svc.ForgotPassword(context.Background(), "notfound@example.com")

	require.NoError(t, err)
	assert.Empty(t, otpRepo.Saved, "no OTP should be created for an unknown email")
	assert.Empty(t, emailSender.Sent)
}

// =============================================================================
// ResetPassword
// =============================================================================

func TestResetPassword_WithValidCode_ResetsPassword(t *testing.T) {
	pwd, _ := valueobjects.NewPassword("oldpass123")
	user := &entities.User{
		ID:           uuid.New(),
		Name:         "Juan",
		Email:        "juan@example.com",
		PasswordHash: pwd.Hash(),
		Locale:       "en",
	}
	otp, code, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposePasswordReset, time.Hour)

	var updatedUser *entities.User
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return otp, nil
		},
	}
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return user, nil
		},
		UpdateFn: func(ctx context.Context, u *entities.User) error {
			updatedUser = u
			return nil
		},
	}

	svc := newTestAuthServiceForRegistration(userRepo, otpRepo, &mocks.MockTokenProvider{}, &mocks.MockEmailSender{})

	err := svc.ResetPassword(context.Background(), "juan@example.com", code, "newpass456")

	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	newPwd := valueobjects.NewPasswordFromHash(updatedUser.PasswordHash)
	assert.NoError(t, newPwd.Compare("newpass456"))
	assert.Error(t, newPwd.Compare("oldpass123"))
}

func TestResetPassword_WithNoOTP_ReturnsInvalidOTP(t *testing.T) {
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return nil, repositories.ErrNotFound
		},
	}

	svc := newTestAuthServiceForRegistration(
		&mocks.MockUserRepository{},
		otpRepo,
		&mocks.MockTokenProvider{},
		&mocks.MockEmailSender{},
	)

	err := svc.ResetPassword(context.Background(), "juan@example.com", "123456", "newpass456")

	assert.ErrorIs(t, err, entities.ErrInvalidOTP)
}

func TestResetPassword_WithWrongCode_IncrementsAttemptsAndReturnsError(t *testing.T) {
	otp, _, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposePasswordReset, time.Hour)
	incremented := false
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return otp, nil
		},
		IncrementAttemptsFn: func(ctx context.Context, id uuid.UUID) error {
			incremented = true
			return nil
		},
	}

	svc := newTestAuthServiceForRegistration(
		&mocks.MockUserRepository{},
		otpRepo,
		&mocks.MockTokenProvider{},
		&mocks.MockEmailSender{},
	)

	err := svc.ResetPassword(context.Background(), "juan@example.com", "000000", "newpass456")

	assert.ErrorIs(t, err, entities.ErrInvalidOTP)
	assert.True(t, incremented, "a failed attempt should be recorded")
}

func TestResetPassword_WithExpiredCode_ReturnsInvalidOTP(t *testing.T) {
	otp, code, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposePasswordReset, time.Hour)
	otp.ExpiresAt = time.Now().Add(-time.Minute)
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return otp, nil
		},
	}

	svc := newTestAuthServiceForRegistration(
		&mocks.MockUserRepository{},
		otpRepo,
		&mocks.MockTokenProvider{},
		&mocks.MockEmailSender{},
	)

	err := svc.ResetPassword(context.Background(), "juan@example.com", code, "newpass456")

	assert.ErrorIs(t, err, entities.ErrInvalidOTP)
}

func TestResetPassword_WithInvalidNewPassword_ReturnsError(t *testing.T) {
	user := &entities.User{
		ID:     uuid.New(),
		Name:   "Juan",
		Email:  "juan@example.com",
		Locale: "en",
	}
	otp, code, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposePasswordReset, time.Hour)
	otpRepo := &mocks.MockOneTimeOTPRepository{
		GetByEmailAndPurposeFn: func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error) {
			return otp, nil
		},
	}
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return user, nil
		},
	}

	svc := newTestAuthServiceForRegistration(userRepo, otpRepo, &mocks.MockTokenProvider{}, &mocks.MockEmailSender{})

	err := svc.ResetPassword(context.Background(), "juan@example.com", code, "short")

	assert.ErrorIs(t, err, valueobjects.ErrPasswordTooShort)
}
