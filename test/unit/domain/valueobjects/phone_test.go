package valueobjects_test

import (
	"testing"

	"ductifact/internal/domain/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPhone_WithValidPhones_ReturnsPhone(t *testing.T) {
	validPhones := []struct {
		name  string
		phone string
	}{
		{"empty (optional)", ""},
		{"international with plus", "+34 612 345 678"},
		{"digits only", "612345678"},
		{"with hyphens", "612-345-678"},
		{"with parentheses", "(612) 345 678"},
		{"US format", "+1 (555) 123-4567"},
		{"short national", "123456"},
	}

	for _, tt := range validPhones {
		t.Run(tt.name, func(t *testing.T) {
			phone, err := valueobjects.NewPhone(tt.phone)

			require.NoError(t, err)
			assert.Equal(t, tt.phone, phone.String())
		})
	}
}

func TestNewPhone_WithInvalidPhones_ReturnsError(t *testing.T) {
	invalidPhones := []struct {
		name  string
		phone string
	}{
		{"too short", "123"},
		{"letters only", "abcdef"},
		{"mixed letters and digits", "abc123def"},
		{"too long", "+12345678901234567890123"},
		{"special characters", "612@345#678"},
	}

	for _, tt := range invalidPhones {
		t.Run(tt.name, func(t *testing.T) {
			phone, err := valueobjects.NewPhone(tt.phone)

			assert.Nil(t, phone)
			assert.ErrorIs(t, err, valueobjects.ErrInvalidPhone)
		})
	}
}
