package entities_test

import (
	"testing"
	"time"

	"ductifact/internal/domain/entities"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOneTimeOTP_ReturnsValidOTPAndCode(t *testing.T) {
	otp, code, err := entities.NewOneTimeOTP("juan@example.com", entities.OTPPurposeRegistration, 15*time.Minute)

	require.NoError(t, err)
	assert.Equal(t, "juan@example.com", otp.Email)
	assert.Equal(t, entities.OTPPurposeRegistration, otp.Purpose)
	assert.Len(t, code, entities.OTPCodeLength)
	assert.Equal(t, 0, otp.Attempts)
	assert.NotEmpty(t, otp.CodeHash)
	assert.NotEqual(t, code, otp.CodeHash, "the plaintext code must never be stored")
	assert.False(t, otp.CreatedAt.IsZero())
	assert.False(t, otp.ExpiresAt.IsZero())
}

func TestNewOneTimeOTP_CodeIsNumeric(t *testing.T) {
	_, code, err := entities.NewOneTimeOTP("a@b.com", entities.OTPPurposePasswordReset, time.Minute)

	require.NoError(t, err)
	for _, c := range code {
		assert.True(t, c >= '0' && c <= '9', "code should only contain digits, got %c", c)
	}
}

func TestNewOneTimeOTP_ExpiresAtMatchesTTL(t *testing.T) {
	before := time.Now()
	otp, _, err := entities.NewOneTimeOTP("a@b.com", entities.OTPPurposeRegistration, 2*time.Hour)
	after := time.Now()

	require.NoError(t, err)
	assert.True(t, otp.ExpiresAt.After(before.Add(2*time.Hour-time.Second)))
	assert.True(t, otp.ExpiresAt.Before(after.Add(2*time.Hour+time.Second)))
}

func TestOneTimeOTP_Verify_WithCorrectCode_ReturnsTrue(t *testing.T) {
	otp, code, _ := entities.NewOneTimeOTP("a@b.com", entities.OTPPurposeRegistration, time.Minute)

	assert.True(t, otp.Verify(code))
}

func TestOneTimeOTP_Verify_WithWrongCode_ReturnsFalse(t *testing.T) {
	otp, _, _ := entities.NewOneTimeOTP("a@b.com", entities.OTPPurposeRegistration, time.Minute)

	assert.False(t, otp.Verify("000000"))
}

func TestOneTimeOTP_IsExpired(t *testing.T) {
	valid, _, _ := entities.NewOneTimeOTP("a@b.com", entities.OTPPurposeRegistration, time.Hour)
	assert.False(t, valid.IsExpired())

	expired := &entities.OneTimeOTP{ExpiresAt: time.Now().Add(-time.Minute)}
	assert.True(t, expired.IsExpired())
}

func TestOneTimeOTP_MaxAttemptsReached(t *testing.T) {
	otp := &entities.OneTimeOTP{Attempts: entities.MaxOTPAttempts - 1}
	assert.False(t, otp.MaxAttemptsReached())

	otp.Attempts = entities.MaxOTPAttempts
	assert.True(t, otp.MaxAttemptsReached())
}

func TestOTPPurpose_Constants(t *testing.T) {
	assert.Equal(t, entities.OTPPurpose("registration"), entities.OTPPurposeRegistration)
	assert.Equal(t, entities.OTPPurpose("password_reset"), entities.OTPPurposePasswordReset)
}
