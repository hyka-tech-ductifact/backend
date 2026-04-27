package email_test

import (
	"testing"

	"ductifact/internal/application/ports"
	"ductifact/internal/infrastructure/adapters/outbound/email"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// BuildMIMEMessage
// =============================================================================

func TestBuildMIMEMessage_MultipartAlternative(t *testing.T) {
	msg := email.BuildMIMEMessage("from@test.com", ports.Email{
		To:      "to@test.com",
		Subject: "Hello",
		HTML:    "<h1>Hi</h1>",
		Text:    "Hi",
	})

	body := string(msg)
	assert.Contains(t, body, "From: from@test.com")
	assert.Contains(t, body, "To: to@test.com")
	assert.Contains(t, body, "Subject: Hello")
	assert.Contains(t, body, "multipart/alternative")
	assert.Contains(t, body, "text/plain")
	assert.Contains(t, body, "text/html")
	assert.Contains(t, body, "<h1>Hi</h1>")
	assert.Contains(t, body, "Hi")
}

func TestBuildMIMEMessage_HTMLOnly(t *testing.T) {
	msg := email.BuildMIMEMessage("from@test.com", ports.Email{
		To:      "to@test.com",
		Subject: "Hello",
		HTML:    "<h1>Hi</h1>",
	})

	body := string(msg)
	assert.Contains(t, body, "Content-Type: text/html")
	assert.NotContains(t, body, "multipart/alternative")
}

func TestBuildMIMEMessage_TextOnly(t *testing.T) {
	msg := email.BuildMIMEMessage("from@test.com", ports.Email{
		To:      "to@test.com",
		Subject: "Hello",
		Text:    "Hi",
	})

	body := string(msg)
	assert.Contains(t, body, "Content-Type: text/plain")
	assert.NotContains(t, body, "multipart/alternative")
}
