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
	DeletedAt    *time.Time // nil = active, non-nil = soft-deleted
}

// NewUser is the only way to create a valid User.
// It validates name and email via setters (single source of truth),
// then hashes the password via the Value Object.
func NewUser(name, email, password string) (*User, error) {
	// Validate and hash the password via the Value Object
	pwd, err := valueobjects.NewPassword(password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	u := &User{
		ID:           uuid.New(),
		PasswordHash: pwd.Hash(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Validate through setters — each setter is the single source of truth.
	if err := u.SetName(name); err != nil {
		return nil, err
	}
	if err := u.SetEmail(email); err != nil {
		return nil, err
	}

	return u, nil
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
