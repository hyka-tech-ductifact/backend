package services

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"ductifact/internal/application/ports"
	"ductifact/internal/application/services/templates"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"
	"ductifact/internal/domain/valueobjects"
)

// --- Application-level errors ---

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrAccountLocked       = errors.New("account temporarily locked due to too many failed login attempts")
)

// authService implements usecases.AuthService.
type authService struct {
	userRepo             repositories.UserRepository
	tokenProvider        ports.TokenProvider
	blacklist            ports.TokenBlacklist
	loginThrottler       ports.LoginThrottler
	emailSender          ports.EmailSender
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo repositories.UserRepository,
	tokenProvider ports.TokenProvider,
	blacklist ports.TokenBlacklist,
	loginThrottler ports.LoginThrottler,
	emailSender ports.EmailSender,
	accessTokenDuration time.Duration,
	refreshTokenDuration time.Duration,
) *authService {
	return &authService{
		userRepo:             userRepo,
		tokenProvider:        tokenProvider,
		blacklist:            blacklist,
		loginThrottler:       loginThrottler,
		emailSender:          emailSender,
		accessTokenDuration:  accessTokenDuration,
		refreshTokenDuration: refreshTokenDuration,
	}
}

// Register creates a new user with a hashed password and returns a token pair.
func (s *authService) Register(ctx context.Context, name, email, password string) (*entities.User, *ports.TokenPair, error) {
	// Step 1: Create user entity (validates name + email + password, hashes password)
	// Done BEFORE the duplicate-email check so that invalid input always
	// returns 400, regardless of whether the email is already taken.
	user, err := entities.NewUser(name, email, password)
	if err != nil {
		return nil, nil, err
	}

	// Step 2: Check if email is already taken
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return nil, nil, err
	}
	if existing != nil {
		return nil, nil, ErrEmailAlreadyInUse
	}

	// Step 3: Persist
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	// Step 4: Generate token pair so the user is logged in immediately
	tokens, err := s.tokenProvider.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		return nil, nil, err
	}

	// Step 5: Send welcome email (non-blocking — registration succeeds even if email fails)
	html, text, err := templates.RenderWelcome(templates.WelcomeData{Name: user.Name})
	if err == nil {
		if err := s.emailSender.Send(ctx, ports.Email{
			To:      user.Email,
			Subject: "Welcome to Ductifact",
			HTML:    html,
			Text:    text,
		}); err != nil {
			slog.Warn("failed to send welcome email", "to", user.Email, "error", err)
		}
	}

	return user, tokens, nil
}

// Login verifies credentials and returns a token pair.
func (s *authService) Login(ctx context.Context, email, password string) (*entities.User, *ports.TokenPair, error) {
	// Step 1: Check if the account is locked due to too many failed attempts
	if s.loginThrottler.IsBlocked(email) {
		return nil, nil, ErrAccountLocked
	}

	// Step 2: Find user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal whether the email exists or not (security)
		s.loginThrottler.RecordFailure(email)
		return nil, nil, ErrInvalidCredentials
	}

	// Step 3: Compare password with stored hash
	pwd := valueobjects.NewPasswordFromHash(user.PasswordHash)
	if err := pwd.Compare(password); err != nil {
		s.loginThrottler.RecordFailure(email)
		return nil, nil, ErrInvalidCredentials
	}

	// Step 4: Login succeeded — clear any previous failures
	s.loginThrottler.Reset(email)

	// Step 5: Generate token pair
	tokens, err := s.tokenProvider.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// RefreshToken validates a refresh token and returns a new token pair.
// This implements JWT rotation: each refresh invalidates the old pair
// and issues a completely new access + refresh token pair.
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*ports.TokenPair, error) {
	// Step 1: Check if the refresh token has been revoked (logout)
	if s.blacklist.IsBlacklisted(refreshToken) {
		return nil, ErrInvalidRefreshToken
	}

	// Step 2: Validate the refresh token
	claims, err := s.tokenProvider.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Step 2: Verify the user still exists (could have been deleted)
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Step 3: Generate a new token pair (rotation)
	tokens, err := s.tokenProvider.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// Logout revokes both the access and refresh tokens by adding them
// to the blacklist. They will remain blacklisted until they naturally expire.
func (s *authService) Logout(_ context.Context, accessToken, refreshToken string) error {
	s.blacklist.Add(accessToken, s.accessTokenDuration)
	s.blacklist.Add(refreshToken, s.refreshTokenDuration)
	return nil
}
