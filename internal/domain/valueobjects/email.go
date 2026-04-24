package valueobjects

import (
	"errors"
	"regexp"
)

var ErrInvalidEmail = errors.New("invalid email format")

// RFC 5321: total max 254 chars, each domain label max 63 chars.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$`)

const maxEmailLength = 254

type Email struct {
	value string
}

func NewEmail(email string) (*Email, error) {
	if len(email) > maxEmailLength || !emailRegex.MatchString(email) {
		return nil, ErrInvalidEmail
	}
	return &Email{value: email}, nil
}

func (e *Email) String() string {
	return e.value
}
