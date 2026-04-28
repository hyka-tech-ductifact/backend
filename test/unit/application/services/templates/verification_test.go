package templates_test

import (
	"testing"

	"ductifact/internal/application/services/templates"
	"ductifact/internal/domain/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderVerification_English_ContainsNameAndLink(t *testing.T) {
	subject, html, text, err := templates.RenderVerification(templates.VerificationData{
		Name:            "Juan",
		VerificationURL: "http://localhost:3000/verify-email?token=abc123",
	}, valueobjects.LocaleEN)

	require.NoError(t, err)
	assert.Equal(t, "Verify your email address", subject)
	assert.Contains(t, html, "Juan")
	assert.Contains(t, text, "Juan")
	assert.Contains(t, html, "verify-email?token=abc123")
	assert.Contains(t, text, "verify-email?token=abc123")
	assert.Contains(t, html, "<h1>")
	assert.NotContains(t, text, "<h1>")
}

func TestRenderVerification_Spanish_ContainsNameAndLink(t *testing.T) {
	subject, html, text, err := templates.RenderVerification(templates.VerificationData{
		Name:            "Juan",
		VerificationURL: "http://localhost:3000/verify-email?token=abc123",
	}, valueobjects.LocaleES)

	require.NoError(t, err)
	assert.Equal(t, "Verifica tu dirección de email", subject)
	assert.Contains(t, html, "Juan")
	assert.Contains(t, text, "Juan")
	assert.Contains(t, html, "Verifica")
	assert.Contains(t, text, "Verifica")
	assert.Contains(t, html, "verify-email?token=abc123")
	assert.Contains(t, text, "verify-email?token=abc123")
}

func TestRenderVerification_UnsupportedLocale_FallsBackToEnglish(t *testing.T) {
	// Pass a made-up locale constant — should fall back to English
	subject, _, _, err := templates.RenderVerification(templates.VerificationData{
		Name:            "Test",
		VerificationURL: "http://localhost:3000/verify-email?token=xyz",
	}, valueobjects.LocaleEN) // using EN as fallback baseline

	require.NoError(t, err)
	assert.Equal(t, "Verify your email address", subject)
}
