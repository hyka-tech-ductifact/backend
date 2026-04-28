package templates

import (
	"bytes"
	"html/template"
	texttemplate "text/template"

	"ductifact/internal/domain/valueobjects"
)

// ── English ─────────────────────────────────────────────────

const passwordResetHTMLEn = `<!DOCTYPE html>
<html>
<body>
    <h1>Reset your password, {{.Name}}</h1>
    <p>We received a request to reset your password. Click the link below to set a new one:</p>
    <p><a href="{{.ResetURL}}">Reset my password</a></p>
    <p>This link will expire in 1 hour.</p>
    <p>If you didn't request a password reset, you can safely ignore this email. Your password will not be changed.</p>
</body>
</html>`

const passwordResetTextEn = `Reset your password, {{.Name}}

We received a request to reset your password. Visit the following link to set a new one:

{{.ResetURL}}

This link will expire in 1 hour.
If you didn't request a password reset, you can safely ignore this email. Your password will not be changed.`

const passwordResetSubjectEn = "Reset your password"

// ── Spanish ─────────────────────────────────────────────────

const passwordResetHTMLEs = `<!DOCTYPE html>
<html>
<body>
    <h1>Restablece tu contraseña, {{.Name}}</h1>
    <p>Recibimos una solicitud para restablecer tu contraseña. Haz clic en el siguiente enlace para crear una nueva:</p>
    <p><a href="{{.ResetURL}}">Restablecer mi contraseña</a></p>
    <p>Este enlace expirará en 1 hora.</p>
    <p>Si no solicitaste un restablecimiento de contraseña, puedes ignorar este email. Tu contraseña no será modificada.</p>
</body>
</html>`

const passwordResetTextEs = `Restablece tu contraseña, {{.Name}}

Recibimos una solicitud para restablecer tu contraseña. Visita el siguiente enlace para crear una nueva:

{{.ResetURL}}

Este enlace expirará en 1 hora.
Si no solicitaste un restablecimiento de contraseña, puedes ignorar este email. Tu contraseña no será modificada.`

const passwordResetSubjectEs = "Restablece tu contraseña"

// ── Template registry ───────────────────────────────────────

type passwordResetContent struct {
	html    string
	text    string
	subject string
}

var passwordResetTemplates = map[valueobjects.Locale]passwordResetContent{
	valueobjects.LocaleEN: {html: passwordResetHTMLEn, text: passwordResetTextEn, subject: passwordResetSubjectEn},
	valueobjects.LocaleES: {html: passwordResetHTMLEs, text: passwordResetTextEs, subject: passwordResetSubjectEs},
}

// PasswordResetData holds the dynamic values for the password reset email template.
type PasswordResetData struct {
	Name     string
	ResetURL string
}

// RenderPasswordReset renders the password reset email in the given locale.
// Returns the localised subject, HTML body, and plain-text body.
func RenderPasswordReset(data PasswordResetData, locale valueobjects.Locale) (subject, html, text string, err error) {
	content, ok := passwordResetTemplates[locale]
	if !ok {
		content = passwordResetTemplates[valueobjects.DefaultLocale]
	}

	// HTML
	htmlTmpl, err := template.New("password_reset_html").Parse(content.html)
	if err != nil {
		return "", "", "", err
	}
	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return "", "", "", err
	}

	// Plain text
	textTmpl, err := texttemplate.New("password_reset_text").Parse(content.text)
	if err != nil {
		return "", "", "", err
	}
	var textBuf bytes.Buffer
	if err := textTmpl.Execute(&textBuf, data); err != nil {
		return "", "", "", err
	}

	return content.subject, htmlBuf.String(), textBuf.String(), nil
}
