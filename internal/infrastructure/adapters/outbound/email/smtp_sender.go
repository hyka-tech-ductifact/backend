package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"

	"ductifact/internal/application/ports"
)

// SMTPSender implements ports.EmailSender using standard SMTP.
// Works with any SMTP provider: SendGrid, SES, Postmark, Mailgun, etc.
type SMTPSender struct {
	host     string // e.g. "smtp.sendgrid.net"
	port     int    // e.g. 587
	useAuth  bool   // whether to use SMTP AUTH
	username string // e.g. "apikey" (SendGrid)
	password string // e.g. your API key
	from     string // e.g. "noreply@ductifact.com"
}

// NewSMTPSender creates a new SMTP-based email sender.
func NewSMTPSender(host string, port int, useAuth bool, username, password, from string) *SMTPSender {
	return &SMTPSender{
		host:     host,
		port:     port,
		useAuth:  useAuth,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPSender) Send(ctx context.Context, email ports.Email) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	var auth smtp.Auth
	if s.useAuth {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	msg := BuildMIMEMessage(s.from, email)

	if err := smtp.SendMail(addr, auth, s.from, []string{email.To}, msg); err != nil {
		return fmt.Errorf("smtp send to %s: %w", email.To, err)
	}
	return nil
}

// Ping checks that the SMTP server is reachable and usable.
//
// It validates SMTP session-level health (EHLO/NOOP) and, when auth is enabled,
// attempts SMTP AUTH to avoid false positives where TCP is reachable but
// credentials are invalid.
func (s *SMTPSender) Ping(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp ping %s: %w", addr, err)
	}
	defer client.Close()

	if err := client.Hello("ductifact-health"); err != nil {
		return fmt.Errorf("smtp hello %s: %w", addr, err)
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: s.host,
			MinVersion: tls.VersionTLS12,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("smtp starttls %s: %w", addr, err)
		}
	}

	if s.useAuth {
		auth := smtp.PlainAuth("", s.username, s.password, s.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth %s: %w", addr, err)
		}
	}

	if err := client.Noop(); err != nil {
		return fmt.Errorf("smtp noop %s: %w", addr, err)
	}

	return nil
}
