package templates

import (
	"bytes"
	"html/template"
	texttemplate "text/template"

	"ductifact/internal/domain/valueobjects"
)

// ── English ─────────────────────────────────────────────────

const welcomeHTMLEn = `<!DOCTYPE html>
<html>
<body>
    <h1>Welcome to Ductifact, {{.Name}}!</h1>
    <p>Your account has been created successfully.</p>
    <p>You can start by creating your first client and project.</p>
</body>
</html>`

const welcomeTextEn = `Welcome to Ductifact, {{.Name}}!

Your account has been created successfully.
You can start by creating your first client and project.`

const welcomeSubjectEn = "Welcome to Ductifact"

// ── Spanish ─────────────────────────────────────────────────

const welcomeHTMLEs = `<!DOCTYPE html>
<html>
<body>
    <h1>¡Bienvenido a Ductifact, {{.Name}}!</h1>
    <p>Tu cuenta ha sido creada correctamente.</p>
    <p>Puedes empezar creando tu primer cliente y proyecto.</p>
</body>
</html>`

const welcomeTextEs = `¡Bienvenido a Ductifact, {{.Name}}!

Tu cuenta ha sido creada correctamente.
Puedes empezar creando tu primer cliente y proyecto.`

const welcomeSubjectEs = "Bienvenido a Ductifact"

// ── Template registry ───────────────────────────────────────

type welcomeContent struct {
	html    string
	text    string
	subject string
}

var welcomeTemplates = map[valueobjects.Locale]welcomeContent{
	valueobjects.LocaleEN: {html: welcomeHTMLEn, text: welcomeTextEn, subject: welcomeSubjectEn},
	valueobjects.LocaleES: {html: welcomeHTMLEs, text: welcomeTextEs, subject: welcomeSubjectEs},
}

// WelcomeData holds the dynamic values for the welcome email template.
type WelcomeData struct {
	Name string
}

// RenderWelcome renders the welcome email in the given locale.
// Returns the localised subject, HTML body, and plain-text body.
func RenderWelcome(data WelcomeData, locale valueobjects.Locale) (subject, html, text string, err error) {
	content, ok := welcomeTemplates[locale]
	if !ok {
		content = welcomeTemplates[valueobjects.DefaultLocale]
	}

	// HTML
	htmlTmpl, err := template.New("welcome_html").Parse(content.html)
	if err != nil {
		return "", "", "", err
	}
	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return "", "", "", err
	}

	// Plain text
	textTmpl, err := texttemplate.New("welcome_text").Parse(content.text)
	if err != nil {
		return "", "", "", err
	}
	var textBuf bytes.Buffer
	if err := textTmpl.Execute(&textBuf, data); err != nil {
		return "", "", "", err
	}

	return content.subject, htmlBuf.String(), textBuf.String(), nil
}
