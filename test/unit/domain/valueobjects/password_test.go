package valueobjects_test

import (
	"testing"

	"ductifact/internal/domain/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPassword_WithValidPasswords_ReturnsPassword(t *testing.T) {
	validPasswords := []struct {
		name     string
		password string
	}{
		{"exactly 8 chars", "12345678"},
		{"long password", "this-is-a-very-secure-password-123"},
		{"with special chars", "p@$$w0rd!"},
		{"with spaces", "pass word with spaces"},
		{"unicode chars", "contraseña123"},
	}

	for _, tt := range validPasswords {
		t.Run(tt.name, func(t *testing.T) {
			pwd, err := valueobjects.NewPassword(tt.password)

			require.NoError(t, err)
			assert.NotEmpty(t, pwd.Hash())
			// The hash should NOT be the raw password
			assert.NotEqual(t, tt.password, pwd.Hash())
		})
	}
}

func TestNewPassword_TooShort_ReturnsError(t *testing.T) {
	shortPasswords := []struct {
		name     string
		password string
	}{
		{"7 chars", "1234567"},
		{"1 char", "a"},
		{"3 chars", "abc"},
	}

	for _, tt := range shortPasswords {
		t.Run(tt.name, func(t *testing.T) {
			pwd, err := valueobjects.NewPassword(tt.password)

			assert.Nil(t, pwd)
			assert.ErrorIs(t, err, valueobjects.ErrPasswordTooShort)
		})
	}
}

func TestNewPassword_Empty_ReturnsError(t *testing.T) {
	pwd, err := valueobjects.NewPassword("")

	assert.Nil(t, pwd)
	assert.ErrorIs(t, err, valueobjects.ErrPasswordEmpty)
}

func TestPassword_Compare_WithCorrectPassword_ReturnsNil(t *testing.T) {
	pwd, _ := valueobjects.NewPassword("securepass123")

	err := pwd.Compare("securepass123")

	assert.NoError(t, err)
}

func TestPassword_Compare_WithWrongPassword_ReturnsError(t *testing.T) {
	pwd, _ := valueobjects.NewPassword("securepass123")

	err := pwd.Compare("wrongpassword")

	assert.ErrorIs(t, err, valueobjects.ErrInvalidPassword)
}

func TestNewPassword_SamePassword_ProducesDifferentHashes(t *testing.T) {
	// bcrypt includes a random salt, so hashing the same password
	// twice should produce different hashes.
	pwd1, _ := valueobjects.NewPassword("securepass123")
	pwd2, _ := valueobjects.NewPassword("securepass123")

	assert.NotEqual(t, pwd1.Hash(), pwd2.Hash())
}

func TestNewPasswordFromHash_AndCompare(t *testing.T) {
	// Simulate: create a password, store the hash, then load it back
	original, _ := valueobjects.NewPassword("securepass123")
	storedHash := original.Hash()

	// Later, load from DB
	loaded := valueobjects.NewPasswordFromHash(storedHash)

	// Should still match the original raw password
	assert.NoError(t, loaded.Compare("securepass123"))
	// Should NOT match a different password
	assert.ErrorIs(t, loaded.Compare("differentpass"), valueobjects.ErrInvalidPassword)
}
