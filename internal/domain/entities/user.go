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
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser is the only way to create a valid User.
// It validates name, email, and password, then hashes the password via the Value Object.
func NewUser(name, email, password string) (*User, error) {
	if name == "" {
		return nil, ErrEmptyUserName
	}

	// Validate email through the value object.
	// The VO is used for validation, but we store the raw string.
	validEmail, err := valueobjects.NewEmail(email)
	if err != nil {
		return nil, err
	}

	// Validate and hash the password via the Value Object
	pwd, err := valueobjects.NewPassword(password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &User{
		ID:           uuid.New(),
		Name:         name,
		Email:        validEmail.String(),
		PasswordHash: pwd.Hash(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// SetName validates and updates the user's name.
func (u *User) SetName(name string) error {
	if name == "" {
		return ErrEmptyUserName
	}
	u.Name = name
	return nil
}

// SetEmail validates format via the Email VO and updates the user's email.
func (u *User) SetEmail(email string) error {
	valid, err := valueobjects.NewEmail(email)
	if err != nil {
		return err
	}
	u.Email = valid.String()
	return nil
}
