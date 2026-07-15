package templates

import (
	"bytes"
	"html/template"
	texttemplate "text/template"

	"ductifact/internal/domain/valueobjects"
)

// ── English ─────────────────────────────────────────────────

const registrationOTPHTMLEn = `<!DOCTYPE html>
<html>
<body>
    <h1>Confirm your email</h1>
    <p>Use the following code to finish creating your Ductifact account:</p>
    <p style="font-size:28px;font-weight:bold;letter-spacing:4px;">{{.Code}}</p>
    <p>This code will expire in {{.ExpiryMinutes}} minutes.</p>
    <p>If you didn't request this, you can safely ignore this email.</p>
</body>
</html>`

const registrationOTPTextEn = `Confirm your email

Use the following code to finish creating your Ductifact account:

{{.Code}}

This code will expire in {{.ExpiryMinutes}} minutes.
If you didn't request this, you can safely ignore this email.`

const registrationOTPSubjectEn = "Your Ductifact verification code"

// ── Spanish ─────────────────────────────────────────────────

const registrationOTPHTMLEs = `<!DOCTYPE html>
<html>
<body>
    <h1>Confirma tu email</h1>
    <p>Usa el siguiente código para terminar de crear tu cuenta de Ductifact:</p>
    <p style="font-size:28px;font-weight:bold;letter-spacing:4px;">{{.Code}}</p>
    <p>Este código expirará en {{.ExpiryMinutes}} minutos.</p>
    <p>Si no solicitaste esto, puedes ignorar este email.</p>
</body>
</html>`

const registrationOTPTextEs = `Confirma tu email

Usa el siguiente código para terminar de crear tu cuenta de Ductifact:

{{.Code}}

Este código expirará en {{.ExpiryMinutes}} minutos.
Si no solicitaste esto, puedes ignorar este email.`

const registrationOTPSubjectEs = "Tu código de verificación de Ductifact"

// ── Template registry ───────────────────────────────────────

type registrationOTPContent struct {
	html    string
	text    string
	subject string
}

var registrationOTPTemplates = map[valueobjects.Locale]registrationOTPContent{
	valueobjects.LocaleEN: {
		html:    registrationOTPHTMLEn,
		text:    registrationOTPTextEn,
		subject: registrationOTPSubjectEn,
	},
	valueobjects.LocaleES: {
		html:    registrationOTPHTMLEs,
		text:    registrationOTPTextEs,
		subject: registrationOTPSubjectEs,
	},
}

// RegistrationOTPData holds the dynamic values for the registration OTP email.
type RegistrationOTPData struct {
	Code          string
	ExpiryMinutes int
}

// RenderRegistrationOTP renders the registration verification-code email in the given locale.
// Returns the localised subject, HTML body, and plain-text body.
func RenderRegistrationOTP(
	data RegistrationOTPData,
	locale valueobjects.Locale,
) (subject, html, text string, err error) {
	content, ok := registrationOTPTemplates[locale]
	if !ok {
		content = registrationOTPTemplates[valueobjects.DefaultLocale]
	}

	htmlTmpl, err := template.New("registration_otp_html").Parse(content.html)
	if err != nil {
		return "", "", "", err
	}
	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return "", "", "", err
	}

	textTmpl, err := texttemplate.New("registration_otp_text").Parse(content.text)
	if err != nil {
		return "", "", "", err
	}
	var textBuf bytes.Buffer
	if err := textTmpl.Execute(&textBuf, data); err != nil {
		return "", "", "", err
	}

	return content.subject, htmlBuf.String(), textBuf.String(), nil
}
