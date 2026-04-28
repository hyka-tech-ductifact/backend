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

	"github.com/google/uuid"
)

// --- Application-level errors ---

var (
	ErrInvalidCredentials       = errors.New("invalid email or password")
	ErrInvalidRefreshToken      = errors.New("invalid or expired refresh token")
	ErrAccountLocked            = errors.New("account temporarily locked due to too many failed login attempts")
	ErrInvalidVerificationToken = errors.New("invalid or expired verification token")
	ErrEmailAlreadyVerified     = errors.New("email already verified")
)

// authService implements usecases.AuthService.
type authService struct {
	userRepo             repositories.UserRepository
	tokenRepo            repositories.OneTimeTokenRepository
	tokenProvider        ports.TokenProvider
	blacklist            ports.TokenBlacklist
	loginThrottler       ports.LoginThrottler
	emailSender          ports.EmailSender
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
	verificationTTL      time.Duration
	frontendURL          string
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo repositories.UserRepository,
	tokenRepo repositories.OneTimeTokenRepository,
	tokenProvider ports.TokenProvider,
	blacklist ports.TokenBlacklist,
	loginThrottler ports.LoginThrottler,
	emailSender ports.EmailSender,
	accessTokenDuration time.Duration,
	refreshTokenDuration time.Duration,
	verificationTTL time.Duration,
	frontendURL string,
) *authService {
	return &authService{
		userRepo:             userRepo,
		tokenRepo:            tokenRepo,
		tokenProvider:        tokenProvider,
		blacklist:            blacklist,
		loginThrottler:       loginThrottler,
		emailSender:          emailSender,
		accessTokenDuration:  accessTokenDuration,
		refreshTokenDuration: refreshTokenDuration,
		verificationTTL:      verificationTTL,
		frontendURL:          frontendURL,
	}
}

// Register creates a new user with a hashed password and returns a token pair.
// If locale is empty, the default ("en") is used.
func (s *authService) Register(ctx context.Context, name, email, password, locale string) (*entities.User, *ports.TokenPair, error) {
	// Apply default locale (application policy, not a domain concern).
	if locale == "" {
		locale = valueobjects.DefaultLocale.String()
	}

	// Step 1: Create user entity (validates name + email + password + locale, hashes password)
	// Done BEFORE the duplicate-email check so that invalid input always
	// returns 400, regardless of whether the email is already taken.
	user, err := entities.NewUser(entities.CreateUserParams{
		Name:     name,
		Email:    email,
		Password: password,
		Locale:   locale,
	})
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

	// Step 5: Create verification token and send welcome email with verification link (non-blocking)
	emailLocale, err := valueobjects.NewLocale(user.Locale)
	if err != nil {
		slog.Error("unexpected invalid locale in user record", "locale", user.Locale, "userID", user.ID, "error", err)
		emailLocale = valueobjects.DefaultLocale
	}
	s.sendWelcomeWithVerification(ctx, user, emailLocale)

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

// VerifyEmail validates the verification token and marks the user's email as verified.
func (s *authService) VerifyEmail(ctx context.Context, token string) error {
	// Step 1: Find the token (scoped to email_verification type)
	vt, err := s.tokenRepo.GetByToken(ctx, token, entities.TokenTypeEmailVerification)
	if err != nil {
		return ErrInvalidVerificationToken
	}

	// Step 2: Check expiration
	if vt.IsExpired() {
		_ = s.tokenRepo.DeleteByUserIDAndType(ctx, vt.UserID, entities.TokenTypeEmailVerification)
		return ErrInvalidVerificationToken
	}

	// Step 3: Get the user
	user, err := s.userRepo.GetByID(ctx, vt.UserID)
	if err != nil {
		return ErrInvalidVerificationToken
	}

	// Step 4: Check if already verified
	if user.IsEmailVerified() {
		_ = s.tokenRepo.DeleteByUserIDAndType(ctx, vt.UserID, entities.TokenTypeEmailVerification)
		return ErrEmailAlreadyVerified
	}

	// Step 5: Mark email as verified
	user.VerifyEmail()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Step 6: Delete all email verification tokens for this user
	_ = s.tokenRepo.DeleteByUserIDAndType(ctx, vt.UserID, entities.TokenTypeEmailVerification)

	return nil
}

// ResendVerificationEmail creates a new verification token and sends the verification email.
// Fails if the user's email is already verified.
func (s *authService) ResendVerificationEmail(ctx context.Context, userID uuid.UUID) error {
	// Step 1: Get the user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	// Step 2: Check if already verified
	if user.IsEmailVerified() {
		return ErrEmailAlreadyVerified
	}

	// Step 3: Resolve locale
	emailLocale, err := valueobjects.NewLocale(user.Locale)
	if err != nil {
		emailLocale = valueobjects.DefaultLocale
	}

	// Step 4: Delete old tokens and send a fresh one
	_ = s.tokenRepo.DeleteByUserIDAndType(ctx, userID, entities.TokenTypeEmailVerification)
	s.sendVerificationEmail(ctx, user, emailLocale)

	return nil
}

// sendWelcomeWithVerification creates a verification token and sends a single welcome
// email that includes both the greeting and the verification link.
// Non-blocking — failures are logged but registration always succeeds.
func (s *authService) sendWelcomeWithVerification(ctx context.Context, user *entities.User, locale valueobjects.Locale) {
	vt, err := entities.NewOneTimeToken(user.ID, entities.TokenTypeEmailVerification, s.verificationTTL)
	if err != nil {
		slog.Error("failed to generate verification token", "userID", user.ID, "error", err)
		return
	}

	if err := s.tokenRepo.Create(ctx, vt); err != nil {
		slog.Error("failed to persist verification token", "userID", user.ID, "error", err)
		return
	}

	verificationURL := s.frontendURL + "/verify-email?token=" + vt.Token
	subject, html, text, err := templates.RenderWelcome(templates.WelcomeData{
		Name:            user.Name,
		VerificationURL: verificationURL,
	}, locale)
	if err != nil {
		slog.Error("failed to render welcome email", "userID", user.ID, "error", err)
		return
	}

	if err := s.emailSender.Send(ctx, ports.Email{
		To:      user.Email,
		Subject: subject,
		HTML:    html,
		Text:    text,
	}); err != nil {
		slog.Warn("failed to send welcome email", "to", user.Email, "error", err)
	}
}

// sendVerificationEmail is a helper that creates a token and sends the verification email.
// Used by resend-verification. Non-blocking — failures are logged but don't break the calling flow.
func (s *authService) sendVerificationEmail(ctx context.Context, user *entities.User, locale valueobjects.Locale) {
	vt, err := entities.NewOneTimeToken(user.ID, entities.TokenTypeEmailVerification, s.verificationTTL)
	if err != nil {
		slog.Error("failed to generate verification token", "userID", user.ID, "error", err)
		return
	}

	if err := s.tokenRepo.Create(ctx, vt); err != nil {
		slog.Error("failed to persist verification token", "userID", user.ID, "error", err)
		return
	}

	verificationURL := s.frontendURL + "/verify-email?token=" + vt.Token
	subject, html, text, err := templates.RenderVerification(templates.VerificationData{
		Name:            user.Name,
		VerificationURL: verificationURL,
	}, locale)
	if err != nil {
		slog.Error("failed to render verification email", "userID", user.ID, "error", err)
		return
	}

	if err := s.emailSender.Send(ctx, ports.Email{
		To:      user.Email,
		Subject: subject,
		HTML:    html,
		Text:    text,
	}); err != nil {
		slog.Warn("failed to send verification email", "to", user.Email, "error", err)
	}
}
