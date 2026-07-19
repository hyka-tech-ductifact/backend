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
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrInvalidRefreshToken    = errors.New("invalid or expired refresh token")
	ErrAccountLocked          = errors.New("account temporarily locked due to too many failed login attempts")
	ErrInvalidCurrentPassword = errors.New("current password is incorrect")
	ErrOTPAlreadyPending      = errors.New("a verification code was already sent and has not yet expired")
)

// authService implements usecases.AuthService.
type authService struct {
	userRepo                  repositories.UserRepository
	otpRepo                   repositories.OneTimeOTPRepository
	tokenProvider             ports.TokenProvider
	blacklist                 ports.TokenBlacklist
	loginThrottler            ports.LoginThrottler
	emailSender               ports.EmailSender
	registrationNoticeLimiter ports.RateLimiter
	accessTokenDuration       time.Duration
	refreshTokenDuration      time.Duration
	registrationOTPTTL        time.Duration
	passwordResetTTL          time.Duration
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo repositories.UserRepository,
	otpRepo repositories.OneTimeOTPRepository,
	tokenProvider ports.TokenProvider,
	blacklist ports.TokenBlacklist,
	loginThrottler ports.LoginThrottler,
	emailSender ports.EmailSender,
	registrationNoticeLimiter ports.RateLimiter,
	accessTokenDuration time.Duration,
	refreshTokenDuration time.Duration,
	registrationOTPTTL time.Duration,
	passwordResetTTL time.Duration,
) *authService {
	return &authService{
		userRepo:                  userRepo,
		otpRepo:                   otpRepo,
		tokenProvider:             tokenProvider,
		blacklist:                 blacklist,
		loginThrottler:            loginThrottler,
		emailSender:               emailSender,
		registrationNoticeLimiter: registrationNoticeLimiter,
		accessTokenDuration:       accessTokenDuration,
		refreshTokenDuration:      refreshTokenDuration,
		registrationOTPTTL:        registrationOTPTTL,
		passwordResetTTL:          passwordResetTTL,
	}
}

// StartRegistration begins the email-first registration flow.
// For a new address, it generates and sends a one-time verification code.
// For an existing account, it sends a rate-limited account-exists notice.
// The caller receives the same result in both cases to prevent enumeration.
func (s *authService) StartRegistration(ctx context.Context, email, locale string) error {
	// Validate the email format via the Value Object.
	validEmail, err := valueobjects.NewEmail(email)
	if err != nil {
		return err
	}
	normalizedEmail := validEmail.String()

	// Resolve locale for the email (application policy).
	if locale == "" {
		locale = valueobjects.DefaultLocale.String()
	}
	emailLocale, err := valueobjects.NewLocale(locale)
	if err != nil {
		emailLocale = valueobjects.DefaultLocale
	}

	// If an account already exists, notify its owner without revealing that fact
	// to the caller. Use the account's saved locale, not attacker-controlled input.
	existing, err := s.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return err
	}
	if existing != nil {
		noticeKey := "registration-account-exists:" + existing.Email
		if s.registrationNoticeLimiter.Allow(noticeKey) {
			emailLocale, localeErr := valueobjects.NewLocale(existing.Locale)
			if localeErr != nil {
				slog.Error(
					"invariant: user has invalid locale",
					"locale", existing.Locale,
					"userID", existing.ID,
					"error", localeErr,
				)
				emailLocale = valueobjects.DefaultLocale
			}
			s.sendAccountAlreadyRegisteredEmail(ctx, existing.Email, emailLocale)
		}
		return nil
	}

	// If a valid (non-expired) OTP already exists, don't spam the user.
	if prev, err := s.otpRepo.GetByEmailAndPurpose(ctx, normalizedEmail, entities.OTPPurposeRegistration); err == nil &&
		prev != nil {
		if !prev.IsExpired() && !prev.MaxAttemptsReached() {
			return ErrOTPAlreadyPending
		}
		// Expired or exhausted — clean it up so the upsert below starts fresh.
		_ = s.otpRepo.DeleteByEmailAndPurpose(ctx, normalizedEmail, entities.OTPPurposeRegistration)
	}

	// Generate a fresh OTP for this email.
	otp, code, err := entities.NewOneTimeOTP(normalizedEmail, entities.OTPPurposeRegistration, s.registrationOTPTTL)
	if err != nil {
		return err
	}
	if err := s.otpRepo.Create(ctx, otp); err != nil {
		return err
	}

	s.sendRegistrationOTP(ctx, normalizedEmail, code, emailLocale)
	return nil
}

// Register completes the email-first registration flow.
// It validates the verification code and, on success, creates the user account
// and returns a token pair so the user is logged in immediately.
func (s *authService) Register(
	ctx context.Context,
	email, code, name, password, locale string,
) (*entities.User, *ports.TokenPair, error) {
	// Apply default locale (application policy, not a domain concern).
	if locale == "" {
		locale = valueobjects.DefaultLocale.String()
	}

	// Normalize the email so the OTP lookup matches how it was stored.
	validEmail, err := valueobjects.NewEmail(email)
	if err != nil {
		return nil, nil, err
	}
	normalizedEmail := validEmail.String()

	// Validate public input before checking the secret code. This keeps password
	// validation independent from OTP validity and avoids creating a code oracle.
	if err := valueobjects.ValidatePassword(password); err != nil {
		return nil, nil, err
	}

	// Step 1: Load the pending OTP for this email.
	otp, err := s.otpRepo.GetByEmailAndPurpose(ctx, normalizedEmail, entities.OTPPurposeRegistration)
	if err != nil {
		return nil, nil, entities.ErrInvalidOTP
	}

	// Step 2: Reject expired or exhausted codes (and clean them up).
	if otp.IsExpired() || otp.MaxAttemptsReached() {
		_ = s.otpRepo.DeleteByEmailAndPurpose(ctx, normalizedEmail, entities.OTPPurposeRegistration)
		return nil, nil, entities.ErrInvalidOTP
	}

	// Step 3: Verify the code. On mismatch, record the failed attempt.
	if !otp.Verify(code) {
		_ = s.otpRepo.IncrementAttempts(ctx, otp.ID)
		return nil, nil, entities.ErrInvalidOTP
	}

	// Step 4: Create the user entity (validates name + email + password + locale, hashes password).
	user, err := entities.NewUser(entities.CreateUserParams{
		Name:     name,
		Email:    normalizedEmail,
		Password: password,
		Locale:   locale,
	})
	if err != nil {
		return nil, nil, err
	}

	// Step 5: Guard against a race where the email was registered meanwhile.
	existing, err := s.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return nil, nil, err
	}
	if existing != nil {
		_ = s.otpRepo.DeleteByEmailAndPurpose(ctx, normalizedEmail, entities.OTPPurposeRegistration)
		return nil, nil, entities.ErrInvalidOTP
	}

	// Step 6: Persist the user.
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	// Step 7: The OTP has served its purpose — remove it.
	_ = s.otpRepo.DeleteByEmailAndPurpose(ctx, normalizedEmail, entities.OTPPurposeRegistration)

	// Step 8: Generate a token pair so the user is logged in immediately.
	tokens, err := s.tokenProvider.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		return nil, nil, err
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

// ChangePassword verifies the current password and updates it to the new one.
// The caller must provide the correct current password for security.
func (s *authService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	// Step 1: Find user by ID
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	// Step 2: Verify current password
	pwd := valueobjects.NewPasswordFromHash(user.PasswordHash)
	if err := pwd.Compare(currentPassword); err != nil {
		return ErrInvalidCurrentPassword
	}

	// Step 3: Validate and hash new password (via entity setter)
	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	// Step 4: Persist updated user
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	return nil
}

// ForgotPassword sends a password reset code to the given address.
// For security, always returns nil even if the email doesn't exist (prevents email enumeration).
func (s *authService) ForgotPassword(ctx context.Context, email string) error {
	// Step 1: Normalize the email so lookups match how it was stored.
	validEmail, err := valueobjects.NewEmail(email)
	if err != nil {
		return err
	}
	normalizedEmail := validEmail.String()

	// Step 2: Find user by email (fail silently if not found).
	user, err := s.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		// Don't reveal whether the email exists
		return nil
	}

	// Step 3: If a valid (non-expired) OTP already exists, don't spam the user.
	if prev, err := s.otpRepo.GetByEmailAndPurpose(ctx, normalizedEmail, entities.OTPPurposePasswordReset); err == nil &&
		prev != nil {
		if !prev.IsExpired() && !prev.MaxAttemptsReached() {
			return nil
		}
		// Expired or exhausted — clean it up so the upsert below starts fresh.
		_ = s.otpRepo.DeleteByEmailAndPurpose(ctx, normalizedEmail, entities.OTPPurposePasswordReset)
	}

	// Step 4: Resolve locale for the email.
	emailLocale, err := valueobjects.NewLocale(user.Locale)
	if err != nil {
		slog.Error("invariant: user has invalid locale", "locale", user.Locale, "userID", user.ID, "error", err)
		emailLocale = valueobjects.DefaultLocale
	}

	// Step 5: Generate a fresh OTP for this email.
	otp, code, err := entities.NewOneTimeOTP(normalizedEmail, entities.OTPPurposePasswordReset, s.passwordResetTTL)
	if err != nil {
		return err
	}
	if err := s.otpRepo.Create(ctx, otp); err != nil {
		return err
	}

	// Step 6: Send the reset code.
	s.sendPasswordResetEmail(ctx, user, code, emailLocale)

	return nil
}

// ResetPassword validates the reset code and sets a new password.
func (s *authService) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	// Step 1: Normalize the email so the OTP lookup matches how it was stored.
	validEmail, err := valueobjects.NewEmail(email)
	if err != nil {
		return entities.ErrInvalidOTP
	}
	normalizedEmail := validEmail.String()

	// Validate public input before checking the secret code so the response does
	// not reveal whether an otherwise invalid request contained a valid OTP.
	if err := valueobjects.ValidatePassword(newPassword); err != nil {
		return err
	}

	// Step 2: Load the pending reset OTP for this email.
	otp, err := s.otpRepo.GetByEmailAndPurpose(ctx, normalizedEmail, entities.OTPPurposePasswordReset)
	if err != nil {
		return entities.ErrInvalidOTP
	}

	// Step 3: Reject expired or exhausted codes (and clean them up).
	if otp.IsExpired() || otp.MaxAttemptsReached() {
		_ = s.otpRepo.DeleteByEmailAndPurpose(ctx, normalizedEmail, entities.OTPPurposePasswordReset)
		return entities.ErrInvalidOTP
	}

	// Step 4: Verify the code. On mismatch, record the failed attempt.
	if !otp.Verify(code) {
		_ = s.otpRepo.IncrementAttempts(ctx, otp.ID)
		return entities.ErrInvalidOTP
	}

	// Step 5: Get the user.
	user, err := s.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		_ = s.otpRepo.DeleteByEmailAndPurpose(ctx, normalizedEmail, entities.OTPPurposePasswordReset)
		return entities.ErrInvalidOTP
	}

	// Step 6: Validate and set new password.
	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	// Step 7: Persist.
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Step 8: The OTP has served its purpose — remove it.
	_ = s.otpRepo.DeleteByEmailAndPurpose(ctx, normalizedEmail, entities.OTPPurposePasswordReset)

	return nil
}

// sendRegistrationOTP renders and sends the verification-code email.
// Non-blocking — failures are logged but don't break the calling flow.
func (s *authService) sendRegistrationOTP(
	ctx context.Context,
	email, code string,
	locale valueobjects.Locale,
) {
	expiryMinutes := int(s.registrationOTPTTL.Minutes())
	subject, html, text, err := templates.RenderRegistrationOTP(templates.RegistrationOTPData{
		Code:          code,
		ExpiryMinutes: expiryMinutes,
	}, locale)
	if err != nil {
		slog.Error("failed to render registration OTP email", "to", email, "error", err)
		return
	}

	if err := s.emailSender.Send(ctx, ports.Email{
		To:      email,
		Subject: subject,
		HTML:    html,
		Text:    text,
	}); err != nil {
		slog.Error("failed to send registration OTP email", "to", email, "error", err)
		return
	}

	slog.Info("registration OTP email sent", "to", email)
}

// sendAccountAlreadyRegisteredEmail tells the address owner how to recover
// access without linking to a specific Ductifact platform.
// Failures are logged but don't alter the generic registration response.
func (s *authService) sendAccountAlreadyRegisteredEmail(
	ctx context.Context,
	email string,
	locale valueobjects.Locale,
) {
	subject, html, text := templates.RenderAccountAlreadyRegistered(locale)

	if err := s.emailSender.Send(ctx, ports.Email{
		To:      email,
		Subject: subject,
		HTML:    html,
		Text:    text,
	}); err != nil {
		slog.Error("failed to send account already registered email", "to", email, "error", err)
		return
	}

	slog.Info("account already registered email sent", "to", email)
}

// sendPasswordResetEmail renders and sends the password reset code email.
// Non-blocking — failures are logged but don't break the calling flow.
func (s *authService) sendPasswordResetEmail(
	ctx context.Context,
	user *entities.User,
	code string,
	locale valueobjects.Locale,
) {
	expiryMinutes := int(s.passwordResetTTL.Minutes())
	subject, html, text, err := templates.RenderPasswordReset(templates.PasswordResetData{
		Name:          user.Name,
		Code:          code,
		ExpiryMinutes: expiryMinutes,
	}, locale)
	if err != nil {
		slog.Error("failed to render password reset email", "userID", user.ID, "error", err)
		return
	}

	if err := s.emailSender.Send(ctx, ports.Email{
		To:      user.Email,
		Subject: subject,
		HTML:    html,
		Text:    text,
	}); err != nil {
		slog.Error("failed to send password reset email", "to", user.Email, "error", err)
		return
	}

	slog.Info("password reset email sent", "to", user.Email)
}
