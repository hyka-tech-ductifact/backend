package usecases

import (
	"context"

	"ductifact/internal/application/ports"
	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// AuthService is the inbound port for authentication operations.
type AuthService interface {
	// StartRegistration begins the email-first registration flow. Its public result
	// is intentionally generic whether it sends a code or an account-exists notice.
	StartRegistration(ctx context.Context, email, locale string) error
	// Register completes registration: it validates the verification code and, on success,
	// creates the user account and returns a token pair (auto-login).
	Register(ctx context.Context, email, code, name, password, locale string) (*entities.User, *ports.TokenPair, error)
	Login(ctx context.Context, email, password string) (*entities.User, *ports.TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*ports.TokenPair, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, email, code, newPassword string) error
}
