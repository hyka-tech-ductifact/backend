package usecases

import (
	"context"

	"ductifact/internal/application/ports"
	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// AuthService is the inbound port for authentication operations.
type AuthService interface {
	Register(ctx context.Context, name, email, password, locale string) (*entities.User, *ports.TokenPair, error)
	Login(ctx context.Context, email, password string) (*entities.User, *ports.TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*ports.TokenPair, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	VerifyEmail(ctx context.Context, token string) error
	ResendVerificationEmail(ctx context.Context, userID uuid.UUID) error
}
