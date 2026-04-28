package templates_test

import (
	"testing"

	"ductifact/internal/application/services/templates"
	"ductifact/internal/domain/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderWelcome_English_ContainsUserName(t *testing.T) {
	subject, html, text, err := templates.RenderWelcome(templates.WelcomeData{
		Name:            "Juan",
		VerificationURL: "http://localhost:3000/verify-email?token=abc123",
	}, valueobjects.LocaleEN)

	require.NoError(t, err)
	assert.Contains(t, subject, "Welcome to Ductifact")
	assert.Contains(t, html, "Juan")
	assert.Contains(t, text, "Juan")
	assert.Contains(t, html, "<h1>")
	assert.NotContains(t, text, "<h1>") // plain text has no HTML tags
	assert.Contains(t, html, "verify-email?token=abc123")
	assert.Contains(t, text, "verify-email?token=abc123")
}

func TestRenderWelcome_Spanish_ContainsUserName(t *testing.T) {
	subject, html, text, err := templates.RenderWelcome(templates.WelcomeData{
		Name:            "Juan",
		VerificationURL: "http://localhost:3000/verify-email?token=abc123",
	}, valueobjects.LocaleES)

	require.NoError(t, err)
	assert.Contains(t, subject, "Bienvenido a Ductifact")
	assert.Contains(t, html, "Juan")
	assert.Contains(t, text, "Juan")
	assert.Contains(t, html, "Bienvenido")
	assert.Contains(t, text, "Bienvenido")
	assert.Contains(t, html, "verify-email?token=abc123")
	assert.Contains(t, text, "verify-email?token=abc123")
}

func TestRenderWelcome_WithEmptyName_Succeeds(t *testing.T) {
	subject, html, text, err := templates.RenderWelcome(templates.WelcomeData{
		Name:            "",
		VerificationURL: "http://localhost:3000/verify-email?token=xyz",
	}, valueobjects.LocaleEN)

	require.NoError(t, err)
	assert.Contains(t, subject, "Welcome to Ductifact")
	assert.Contains(t, html, "Welcome to Ductifact")
	assert.Contains(t, text, "Welcome to Ductifact")
}
