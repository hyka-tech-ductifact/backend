package mocks

import (
	"context"

	"ductifact/internal/application/ports"
)

// MockEmailSender implements ports.EmailSender for testing.
type MockEmailSender struct {
	SendFn func(ctx context.Context, email ports.Email) error
	PingFn func(ctx context.Context) error
	Sent   []ports.Email // captures all emails sent (when SendFn is nil)
}

func (m *MockEmailSender) Send(ctx context.Context, email ports.Email) error {
	if m.SendFn != nil {
		return m.SendFn(ctx, email)
	}
	m.Sent = append(m.Sent, email)
	return nil
}

func (m *MockEmailSender) Ping(ctx context.Context) error {
	if m.PingFn != nil {
		return m.PingFn(ctx)
	}
	return nil
}
