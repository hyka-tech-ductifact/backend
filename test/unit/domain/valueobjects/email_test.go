package valueobjects_test

import (
	"testing"

	"ductifact/internal/domain/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmail_WithValidEmails_ReturnsEmail(t *testing.T) {
	validEmails := []struct {
		name  string
		email string
	}{
		{"simple email", "user@example.com"},
		{"with dots", "first.last@example.com"},
		{"with plus", "user+tag@example.com"},
		{"with subdomain", "user@mail.example.com"},
		{"with numbers", "user123@example.com"},
		{"with percent", "user%name@example.com"},
		{"with hyphen in domain", "user@my-domain.com"},
	}

	for _, tt := range validEmails {
		t.Run(tt.name, func(t *testing.T) {
			email, err := valueobjects.NewEmail(tt.email)

			require.NoError(t, err)
			assert.Equal(t, tt.email, email.String())
		})
	}
}

func TestNewEmail_WithInvalidEmails_ReturnsError(t *testing.T) {
	invalidEmails := []struct {
		name  string
		email string
	}{
		{"empty string", ""},
		{"no at sign", "userexample.com"},
		{"no domain", "user@"},
		{"no local part", "@example.com"},
		{"spaces in local", "user @example.com"},
		{"double at", "user@@example.com"},
		{"no TLD", "user@example"},
		{"only whitespace", "   "},
		{"missing domain after at", "user@.com"},
		{"domain label ends with hyphen", "user@example-.com"},
		{"domain label starts with hyphen", "user@-example.com"},
		{"domain label with only hyphen", "_0AncD@9nkNy0.Q-.Lf"},
	}

	for _, tt := range invalidEmails {
		t.Run(tt.name, func(t *testing.T) {
			email, err := valueobjects.NewEmail(tt.email)

			assert.Nil(t, email)
			assert.ErrorIs(t, err, valueobjects.ErrInvalidEmail)
		})
	}
}
