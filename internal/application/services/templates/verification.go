package templates

import (
	"bytes"
	"html/template"
	texttemplate "text/template"

	"ductifact/internal/domain/valueobjects"
)

// ── English ─────────────────────────────────────────────────

const verificationHTMLEn = `<!DOCTYPE html>
<html>
<body>
    <h1>Verify your email, {{.Name}}</h1>
    <p>Please click the link below to verify your email address:</p>
    <p><a href="{{.VerificationURL}}">Verify my email</a></p>
    <p>This link will expire in 24 hours.</p>
    <p>If you didn't create an account on Ductifact, you can ignore this email.</p>
</body>
</html>`

const verificationTextEn = `Verify your email, {{.Name}}

Please visit the following link to verify your email address:

{{.VerificationURL}}

This link will expire in 24 hours.
If you didn't create an account on Ductifact, you can ignore this email.`

const verificationSubjectEn = "Verify your email address"

// ── Spanish ─────────────────────────────────────────────────

const verificationHTMLEs = `<!DOCTYPE html>
<html>
<body>
    <h1>Verifica tu email, {{.Name}}</h1>
    <p>Haz clic en el siguiente enlace para verificar tu dirección de email:</p>
    <p><a href="{{.VerificationURL}}">Verificar mi email</a></p>
    <p>Este enlace expirará en 24 horas.</p>
    <p>Si no creaste una cuenta en Ductifact, puedes ignorar este email.</p>
</body>
</html>`

const verificationTextEs = `Verifica tu email, {{.Name}}

Visita el siguiente enlace para verificar tu dirección de email:

{{.VerificationURL}}

Este enlace expirará en 24 horas.
Si no creaste una cuenta en Ductifact, puedes ignorar este email.`

const verificationSubjectEs = "Verifica tu dirección de email"

// ── Template registry ───────────────────────────────────────

type verificationContent struct {
	html    string
	text    string
	subject string
}

var verificationTemplates = map[valueobjects.Locale]verificationContent{
	valueobjects.LocaleEN: {html: verificationHTMLEn, text: verificationTextEn, subject: verificationSubjectEn},
	valueobjects.LocaleES: {html: verificationHTMLEs, text: verificationTextEs, subject: verificationSubjectEs},
}

// VerificationData holds the dynamic values for the email verification template.
type VerificationData struct {
	Name            string
	VerificationURL string
}

// RenderVerification renders the email verification email in the given locale.
// Returns the localised subject, HTML body, and plain-text body.
func RenderVerification(data VerificationData, locale valueobjects.Locale) (subject, html, text string, err error) {
	content, ok := verificationTemplates[locale]
	if !ok {
		content = verificationTemplates[valueobjects.DefaultLocale]
	}

	// HTML
	htmlTmpl, err := template.New("verification_html").Parse(content.html)
	if err != nil {
		return "", "", "", err
	}
	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return "", "", "", err
	}

	// Plain text
	textTmpl, err := texttemplate.New("verification_text").Parse(content.text)
	if err != nil {
		return "", "", "", err
	}
	var textBuf bytes.Buffer
	if err := textTmpl.Execute(&textBuf, data); err != nil {
		return "", "", "", err
	}

	return content.subject, htmlBuf.String(), textBuf.String(), nil
}
