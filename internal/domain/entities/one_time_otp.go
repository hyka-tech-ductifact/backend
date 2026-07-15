package entities

import (
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// OTPCodeLength is the number of digits in a one-time verification code.
const OTPCodeLength = 6

// MaxOTPAttempts is the number of failed verification attempts allowed
// before the code is invalidated (online brute-force protection).
const MaxOTPAttempts = 5

var (
	// ErrInvalidOTP is returned when a verification code is wrong, expired, or missing.
	ErrInvalidOTP = errors.New("invalid or expired verification code")
)

// OTPPurpose identifies what a one-time code is used for.
type OTPPurpose string

const (
	// OTPPurposeRegistration proves ownership of an email before account creation.
	OTPPurposeRegistration OTPPurpose = "registration"
	// OTPPurposePasswordReset authorizes setting a new password for an existing account.
	OTPPurposePasswordReset OTPPurpose = "password_reset"
)

// OneTimeOTP is a short-lived, single-use numeric code tied to an email address.
// The Purpose field determines its use (registration, password reset, ...).
// It stores only the bcrypt hash of the code, never the plaintext.
type OneTimeOTP struct {
	ID        uuid.UUID
	Email     string
	Purpose   OTPPurpose
	CodeHash  string
	ExpiresAt time.Time
	Attempts  int
	CreatedAt time.Time
}

// NewOneTimeOTP generates a fresh OTP for the given email and purpose.
// It returns the entity (with the hashed code) and the plaintext code,
// which must be delivered to the user and never persisted.
func NewOneTimeOTP(email string, purpose OTPPurpose, ttl time.Duration) (*OneTimeOTP, string, error) {
	code, err := generateNumericCode(OTPCodeLength)
	if err != nil {
		return nil, "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	otp := &OneTimeOTP{
		ID:        uuid.New(),
		Email:     email,
		Purpose:   purpose,
		CodeHash:  string(hash),
		ExpiresAt: now.Add(ttl),
		Attempts:  0,
		CreatedAt: now,
	}
	return otp, code, nil
}

// IsExpired returns true if the code has passed its expiration time.
func (o *OneTimeOTP) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}

// MaxAttemptsReached returns true if too many failed attempts have been made.
func (o *OneTimeOTP) MaxAttemptsReached() bool {
	return o.Attempts >= MaxOTPAttempts
}

// Verify checks the supplied code against the stored hash (constant-time via bcrypt).
func (o *OneTimeOTP) Verify(code string) bool {
	return bcrypt.CompareHashAndPassword([]byte(o.CodeHash), []byte(code)) == nil
}

// generateNumericCode returns a cryptographically-random decimal string of n digits,
// preserving leading zeros (e.g. "004217").
func generateNumericCode(n int) (string, error) {
	digits := make([]byte, n)
	for i := 0; i < n; i++ {
		d, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		digits[i] = byte('0' + d.Int64())
	}
	return string(digits), nil
}
