package templates

import "ductifact/internal/domain/valueobjects"

// ── English ─────────────────────────────────────────────────

const accountAlreadyRegisteredHTMLEn = `<!DOCTYPE html>
<html>
<body>
    <h1>You already have a Ductifact account</h1>
    <p>We received a request to create an account using this email address.</p>
    <p>An account is already registered with this address.</p>
    <p>If you have forgotten your password, open Ductifact on your preferred platform, web or app, go to password recovery, and request a reset code.</p>
    <p>If you didn't make this request, you can safely ignore this email. Your password has not been changed.</p>
</body>
</html>`

const accountAlreadyRegisteredTextEn = `You already have a Ductifact account

We received a request to create an account using this email address.
An account is already registered with this address.

If you have forgotten your password, open Ductifact on your preferred platform, web or app, go to password recovery, and request a reset code.

If you didn't make this request, you can safely ignore this email. Your password has not been changed.`

const accountAlreadyRegisteredSubjectEn = "You already have a Ductifact account"

// ── Spanish ─────────────────────────────────────────────────

const accountAlreadyRegisteredHTMLEs = `<!DOCTYPE html>
<html>
<body>
    <h1>Ya tienes una cuenta en Ductifact</h1>
    <p>Hemos recibido una solicitud para crear una cuenta con esta dirección de email.</p>
    <p>Ya existe una cuenta registrada con esta dirección.</p>
    <p>Si has olvidado tu contraseña, abre Ductifact en la plataforma que prefieras, web o aplicación, ve a la recuperación de contraseña y solicita un código de restablecimiento.</p>
    <p>Si no realizaste esta solicitud, puedes ignorar este email. Tu contraseña no ha sido modificada.</p>
</body>
</html>`

const accountAlreadyRegisteredTextEs = `Ya tienes una cuenta en Ductifact

Hemos recibido una solicitud para crear una cuenta con esta dirección de email.
Ya existe una cuenta registrada con esta dirección.

Si has olvidado tu contraseña, abre Ductifact en la plataforma que prefieras, web o aplicación, ve a la recuperación de contraseña y solicita un código de restablecimiento.

Si no realizaste esta solicitud, puedes ignorar este email. Tu contraseña no ha sido modificada.`

const accountAlreadyRegisteredSubjectEs = "Ya tienes una cuenta en Ductifact"

// ── Template registry ───────────────────────────────────────

type accountAlreadyRegisteredContent struct {
	html    string
	text    string
	subject string
}

var accountAlreadyRegisteredTemplates = map[valueobjects.Locale]accountAlreadyRegisteredContent{
	valueobjects.LocaleEN: {
		html:    accountAlreadyRegisteredHTMLEn,
		text:    accountAlreadyRegisteredTextEn,
		subject: accountAlreadyRegisteredSubjectEn,
	},
	valueobjects.LocaleES: {
		html:    accountAlreadyRegisteredHTMLEs,
		text:    accountAlreadyRegisteredTextEs,
		subject: accountAlreadyRegisteredSubjectEs,
	},
}

// RenderAccountAlreadyRegistered returns the localized account-exists notice.
func RenderAccountAlreadyRegistered(locale valueobjects.Locale) (subject, html, text string) {
	content, ok := accountAlreadyRegisteredTemplates[locale]
	if !ok {
		content = accountAlreadyRegisteredTemplates[valueobjects.DefaultLocale]
	}

	return content.subject, content.html, content.text
}
