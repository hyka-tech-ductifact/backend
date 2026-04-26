package templates_test

import (
	"testing"

	"ductifact/internal/application/services/templates"
	"ductifact/internal/domain/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderWelcome_English_ContainsUserName(t *testing.T) {
	subject, html, text, err := templates.RenderWelcome(templates.WelcomeData{Name: "Juan"}, valueobjects.LocaleEN)

	require.NoError(t, err)
	assert.Equal(t, "Welcome to Ductifact", subject)
	assert.Contains(t, html, "Juan")
	assert.Contains(t, text, "Juan")
	assert.Contains(t, html, "<h1>")
	assert.NotContains(t, text, "<h1>") // plain text has no HTML tags
}

func TestRenderWelcome_Spanish_ContainsUserName(t *testing.T) {
	subject, html, text, err := templates.RenderWelcome(templates.WelcomeData{Name: "Juan"}, valueobjects.LocaleES)

	require.NoError(t, err)
	assert.Equal(t, "Bienvenido a Ductifact", subject)
	assert.Contains(t, html, "Juan")
	assert.Contains(t, text, "Juan")
	assert.Contains(t, html, "Bienvenido")
	assert.Contains(t, text, "Bienvenido")
}

func TestRenderWelcome_WithEmptyName_Succeeds(t *testing.T) {
	subject, html, text, err := templates.RenderWelcome(templates.WelcomeData{Name: ""}, valueobjects.LocaleEN)

	require.NoError(t, err)
	assert.Equal(t, "Welcome to Ductifact", subject)
	assert.Contains(t, html, "Welcome to Ductifact")
	assert.Contains(t, text, "Welcome to Ductifact")
}
