package ports

import "context"

// Email represents an email message to be sent.
type Email struct {
	To      string // recipient email address
	Subject string
	HTML    string // rendered HTML body
	Text    string // plain text fallback (for clients that don't support HTML)
}

// EmailSender is the outbound port for sending transactional emails.
// The implementation (SMTP, SendGrid, console logger, etc.) lives in infrastructure.
type EmailSender interface {
	// Send delivers a transactional email to the recipient.
	// Returns an error if the delivery fails.
	Send(ctx context.Context, email Email) error

	// Ping checks whether the email backend is reachable.
	// Used by the readiness probe to verify email health.
	Ping(ctx context.Context) error
}
