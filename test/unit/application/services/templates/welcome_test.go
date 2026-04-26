package templates_test

import (
	"testing"

	"ductifact/internal/application/services/templates"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderWelcome_ContainsUserName(t *testing.T) {
	html, text, err := templates.RenderWelcome(templates.WelcomeData{Name: "Juan"})

	require.NoError(t, err)
	assert.Contains(t, html, "Juan")
	assert.Contains(t, text, "Juan")
	assert.Contains(t, html, "<h1>")
	assert.NotContains(t, text, "<h1>") // plain text has no HTML tags
}

func TestRenderWelcome_WithEmptyName_Succeeds(t *testing.T) {
	html, text, err := templates.RenderWelcome(templates.WelcomeData{Name: ""})

	require.NoError(t, err)
	assert.Contains(t, html, "Welcome to Ductifact")
	assert.Contains(t, text, "Welcome to Ductifact")
}
