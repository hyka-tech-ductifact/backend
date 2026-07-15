package mocks

import (
	"context"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// MockOneTimeOTPRepository implements repositories.OneTimeOTPRepository for testing.
type MockOneTimeOTPRepository struct {
	CreateFn                  func(ctx context.Context, otp *entities.OneTimeOTP) error
	GetByEmailAndPurposeFn    func(ctx context.Context, email string, purpose entities.OTPPurpose) (*entities.OneTimeOTP, error)
	IncrementAttemptsFn       func(ctx context.Context, id uuid.UUID) error
	DeleteByEmailAndPurposeFn func(ctx context.Context, email string, purpose entities.OTPPurpose) error
	Created                   []*entities.OneTimeOTP // captures all OTPs created (when CreateFn is nil)
	Saved                     []*entities.OneTimeOTP // backward-compatible alias used by existing tests
}

func (m *MockOneTimeOTPRepository) Create(ctx context.Context, otp *entities.OneTimeOTP) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, otp)
	}
	m.Created = append(m.Created, otp)
	m.Saved = append(m.Saved, otp)
	return nil
}

func (m *MockOneTimeOTPRepository) GetByEmailAndPurpose(
	ctx context.Context,
	email string,
	purpose entities.OTPPurpose,
) (*entities.OneTimeOTP, error) {
	if m.GetByEmailAndPurposeFn != nil {
		return m.GetByEmailAndPurposeFn(ctx, email, purpose)
	}
	return nil, nil
}

func (m *MockOneTimeOTPRepository) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	if m.IncrementAttemptsFn != nil {
		return m.IncrementAttemptsFn(ctx, id)
	}
	return nil
}

func (m *MockOneTimeOTPRepository) DeleteByEmailAndPurpose(
	ctx context.Context,
	email string,
	purpose entities.OTPPurpose,
) error {
	if m.DeleteByEmailAndPurposeFn != nil {
		return m.DeleteByEmailAndPurposeFn(ctx, email, purpose)
	}
	return nil
}
