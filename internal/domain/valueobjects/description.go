package valueobjects

import (
	"errors"
)

var (
	ErrDescriptionTooLong = errors.New("description must not exceed 500 characters")
)

const MaxDescriptionLength = 500

// Description is a value object that validates description text.
type Description struct {
	value string
}

// NewDescription validates length constraints and returns a Description.
// An empty description is considered valid (description is optional).
func NewDescription(desc string) (*Description, error) {
	if len(desc) > MaxDescriptionLength {
		return nil, ErrDescriptionTooLong
	}
	return &Description{value: desc}, nil
}

func (d *Description) String() string {
	return d.value
}
