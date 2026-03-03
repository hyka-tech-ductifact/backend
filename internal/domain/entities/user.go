package entities

import (
	"ductifact/internal/domain/valueobjects"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyUserName = errors.New("user name cannot be empty")
)

type User struct {
	ID        uuid.UUID
	Name      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewUser is the only way to create a valid User.
// It validates all business rules and returns an error if any fail.
func NewUser(name, email string) (*User, error) {
	if name == "" {
		return nil, ErrEmptyUserName
	}
	// Validate email through the value object.
	// The VO is used for validation, but we store the raw string.
	// This avoids coupling the entity struct to the VO type.
	validEmail, err := valueobjects.NewEmail(email)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &User{
		ID:        uuid.New(),
		Name:      name,
		Email:     validEmail.String(),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
