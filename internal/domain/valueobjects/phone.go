package valueobjects

import (
	"errors"
	"regexp"
)

var (
	ErrInvalidPhone = errors.New("invalid phone format")
)

// phoneRegex accepts international formats: optional +, digits, spaces, hyphens, parentheses.
// Requires at least 6 digits total (to support short national numbers).
var phoneRegex = regexp.MustCompile(`^\+?[\d\s\-()]{6,20}$`)

// Phone is a value object that validates phone numbers.
type Phone struct {
	value string
}

// NewPhone validates the phone format and returns a Phone.
// An empty phone is considered valid (phone is optional).
func NewPhone(phone string) (*Phone, error) {
	if phone == "" {
		return &Phone{value: ""}, nil
	}
	if !phoneRegex.MatchString(phone) {
		return nil, ErrInvalidPhone
	}
	return &Phone{value: phone}, nil
}

func (p *Phone) String() string {
	return p.value
}
