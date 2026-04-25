package email

import (
	"fmt"
	"strings"

	"ductifact/internal/application/ports"
)

// BuildMIMEMessage constructs a multipart MIME email with HTML and plain text parts.
// If only HTML is provided, a simple HTML email is built.
// If both HTML and Text are provided, a multipart/alternative message is built
// so email clients can choose which version to display.
func BuildMIMEMessage(from string, email ports.Email) []byte {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("From: %s\r\n", from))
	b.WriteString(fmt.Sprintf("To: %s\r\n", email.To))
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	b.WriteString("MIME-Version: 1.0\r\n")

	if email.Text != "" && email.HTML != "" {
		// Multipart alternative: includes both text and HTML
		boundary := "ductifact-boundary-0001"
		b.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", boundary))
		b.WriteString("\r\n")

		// Plain text part
		b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		b.WriteString("\r\n")
		b.WriteString(email.Text)
		b.WriteString("\r\n")

		// HTML part
		b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		b.WriteString("\r\n")
		b.WriteString(email.HTML)
		b.WriteString("\r\n")

		b.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if email.HTML != "" {
		// HTML only
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		b.WriteString("\r\n")
		b.WriteString(email.HTML)
	} else {
		// Plain text only
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		b.WriteString("\r\n")
		b.WriteString(email.Text)
	}

	return []byte(b.String())
}
