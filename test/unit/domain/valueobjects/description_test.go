package valueobjects_test

import (
	"strings"
	"testing"

	"ductifact/internal/domain/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDescription_WithValidDescriptions_ReturnsDescription(t *testing.T) {
	validDescs := []struct {
		name string
		desc string
	}{
		{"empty (optional)", ""},
		{"short text", "Main partner"},
		{"max length", strings.Repeat("a", valueobjects.MaxDescriptionLength)},
		{"with special chars", "Client description: café & más!"},
	}

	for _, tt := range validDescs {
		t.Run(tt.name, func(t *testing.T) {
			desc, err := valueobjects.NewDescription(tt.desc)

			require.NoError(t, err)
			assert.Equal(t, tt.desc, desc.String())
		})
	}
}

func TestNewDescription_TooLong_ReturnsError(t *testing.T) {
	tooLong := strings.Repeat("a", valueobjects.MaxDescriptionLength+1)

	desc, err := valueobjects.NewDescription(tooLong)

	assert.Nil(t, desc)
	assert.ErrorIs(t, err, valueobjects.ErrDescriptionTooLong)
}
