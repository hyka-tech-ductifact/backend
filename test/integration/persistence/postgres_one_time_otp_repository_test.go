package persistence_test

import (
	"context"
	"testing"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"
	"ductifact/internal/infrastructure/adapters/outbound/persistence"
	"ductifact/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupOneTimeOTPRepo(t *testing.T) *persistence.PostgresOneTimeOTPRepository {
	db := helpers.SetupTestDB(t)
	helpers.CleanDB(t, db)
	return persistence.NewPostgresOneTimeOTPRepository(db)
}

func TestOneTimeOTPRepository_Save_And_GetByEmailAndPurpose(t *testing.T) {
	repo := setupOneTimeOTPRepo(t)
	ctx := context.Background()

	otp, _, err := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, otp))

	found, err := repo.GetByEmailAndPurpose(ctx, "juan@example.com", entities.OTPPurposeRegistration)
	require.NoError(t, err)
	assert.Equal(t, otp.ID, found.ID)
	assert.Equal(t, "juan@example.com", found.Email)
	assert.Equal(t, entities.OTPPurposeRegistration, found.Purpose)
	assert.Equal(t, otp.CodeHash, found.CodeHash)
}

func TestOneTimeOTPRepository_Save_ReplacesExistingForSameEmailAndPurpose(t *testing.T) {
	repo := setupOneTimeOTPRepo(t)
	ctx := context.Background()

	first, _, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	require.NoError(t, repo.Create(ctx, first))

	second, _, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	require.NoError(t, repo.Create(ctx, second))

	found, err := repo.GetByEmailAndPurpose(ctx, "juan@example.com", entities.OTPPurposeRegistration)
	require.NoError(t, err)
	assert.Equal(t, second.ID, found.ID, "the newer OTP should replace the previous one")
}

func TestOneTimeOTPRepository_DifferentPurposesCoexist(t *testing.T) {
	repo := setupOneTimeOTPRepo(t)
	ctx := context.Background()

	reg, _, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	reset, _, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposePasswordReset, 15*time.Minute)
	require.NoError(t, repo.Create(ctx, reg))
	require.NoError(t, repo.Create(ctx, reset))

	foundReg, err := repo.GetByEmailAndPurpose(ctx, "juan@example.com", entities.OTPPurposeRegistration)
	require.NoError(t, err)
	assert.Equal(t, reg.ID, foundReg.ID)

	foundReset, err := repo.GetByEmailAndPurpose(ctx, "juan@example.com", entities.OTPPurposePasswordReset)
	require.NoError(t, err)
	assert.Equal(t, reset.ID, foundReset.ID)
}

func TestOneTimeOTPRepository_GetByEmailAndPurpose_NotFound(t *testing.T) {
	repo := setupOneTimeOTPRepo(t)

	_, err := repo.GetByEmailAndPurpose(context.Background(), "nobody@example.com", entities.OTPPurposeRegistration)
	assert.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestOneTimeOTPRepository_IncrementAttempts(t *testing.T) {
	repo := setupOneTimeOTPRepo(t)
	ctx := context.Background()

	otp, _, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	require.NoError(t, repo.Create(ctx, otp))

	require.NoError(t, repo.IncrementAttempts(ctx, otp.ID))
	require.NoError(t, repo.IncrementAttempts(ctx, otp.ID))

	found, err := repo.GetByEmailAndPurpose(ctx, "juan@example.com", entities.OTPPurposeRegistration)
	require.NoError(t, err)
	assert.Equal(t, 2, found.Attempts)
}

func TestOneTimeOTPRepository_DeleteByEmailAndPurpose(t *testing.T) {
	repo := setupOneTimeOTPRepo(t)
	ctx := context.Background()

	otp, _, _ := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)
	require.NoError(t, repo.Create(ctx, otp))

	require.NoError(t, repo.DeleteByEmailAndPurpose(ctx, "juan@example.com", entities.OTPPurposeRegistration))

	_, err := repo.GetByEmailAndPurpose(ctx, "juan@example.com", entities.OTPPurposeRegistration)
	assert.ErrorIs(t, err, repositories.ErrNotFound)
}
