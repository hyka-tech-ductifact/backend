package valueobjects_test

import (
	"testing"

	"ductifact/internal/domain/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLocale_WithEnglish_ReturnsLocale(t *testing.T) {
	locale, err := valueobjects.NewLocale("en")

	require.NoError(t, err)
	assert.Equal(t, "en", locale.String())
}

func TestNewLocale_WithSpanish_ReturnsLocale(t *testing.T) {
	locale, err := valueobjects.NewLocale("es")

	require.NoError(t, err)
	assert.Equal(t, "es", locale.String())
}

func TestNewLocale_WithUnsupported_ReturnsError(t *testing.T) {
	_, err := valueobjects.NewLocale("fr")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid locale")
	assert.Contains(t, err.Error(), "en")
	assert.Contains(t, err.Error(), "es")
}

func TestNewLocale_WithEmpty_ReturnsError(t *testing.T) {
	_, err := valueobjects.NewLocale("")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid locale")
}

func TestSupportedLocales_ReturnsAllLocales(t *testing.T) {
	locales := valueobjects.SupportedLocales()

	assert.Contains(t, locales, "en")
	assert.Contains(t, locales, "es")
	assert.Len(t, locales, 2)
}

func TestDefaultLocale_IsEnglish(t *testing.T) {
	assert.Equal(t, "en", valueobjects.DefaultLocale.String())
}
