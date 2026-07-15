package repositories

import (
	"context"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// OneTimeOTPRepository is the outbound port for one-time OTP persistence.
// There is at most one OTP per (email, purpose); creating a new one replaces the previous.
type OneTimeOTPRepository interface {
	// Create inserts the OTP, replacing any existing OTP for the same (email, purpose).
	Create(ctx context.Context, otp *entities.OneTimeOTP) error
	GetByEmailAndPurpose(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error)
	// IncrementAttempts atomically increases the failed-attempt counter.
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
	DeleteByEmailAndPurpose(ctx context.Context, email string, purpose entities.OTPPurpose) error
}
