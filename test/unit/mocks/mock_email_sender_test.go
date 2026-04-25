package mocks_test

import (
	"context"
	"errors"
	"testing"

	"ductifact/internal/application/ports"
	"ductifact/test/unit/mocks"

	"github.com/stretchr/testify/assert"
)

func TestMockEmailSender_CapturesEmails(t *testing.T) {
	mock := &mocks.MockEmailSender{}

	_ = mock.Send(context.Background(), ports.Email{
		To:      "a@test.com",
		Subject: "First",
	})
	_ = mock.Send(context.Background(), ports.Email{
		To:      "b@test.com",
		Subject: "Second",
	})

	assert.Len(t, mock.Sent, 2)
	assert.Equal(t, "a@test.com", mock.Sent[0].To)
	assert.Equal(t, "Second", mock.Sent[1].Subject)
}

func TestMockEmailSender_SendFn_OverridesDefault(t *testing.T) {
	expectedErr := errors.New("smtp down")
	mock := &mocks.MockEmailSender{
		SendFn: func(ctx context.Context, email ports.Email) error {
			return expectedErr
		},
	}

	err := mock.Send(context.Background(), ports.Email{To: "x@test.com"})

	assert.ErrorIs(t, err, expectedErr)
	assert.Empty(t, mock.Sent) // SendFn was used, not the default capture
}
