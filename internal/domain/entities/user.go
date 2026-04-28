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
	ID              uuid.UUID
	Name            string
	Email           string
	PasswordHash    string
	Locale          string     // BCP 47 language tag (e.g. "en", "es")
	EmailVerifiedAt *time.Time // nil = unverified
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time // nil = active, non-nil = soft-deleted
}

// CreateUserParams groups all parameters needed to create a User.
// Required: Name, Email, Password, Locale.
type CreateUserParams struct {
	Name     string
	Email    string
	Password string
	Locale   string
}

// NewUser is the only way to create a valid User.
// It validates name, email, password and locale via their respective Value Objects.
// All fields are required — the caller must supply a valid locale.
func NewUser(params CreateUserParams) (*User, error) {
	// Validate and hash the password via the Value Object
	pwd, err := valueobjects.NewPassword(params.Password)
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
	if err := u.SetName(params.Name); err != nil {
		return nil, err
	}
	if err := u.SetEmail(params.Email); err != nil {
		return nil, err
	}
	if err := u.SetLocale(params.Locale); err != nil {
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

// SetLocale validates and updates the user's preferred locale.
func (u *User) SetLocale(locale string) error {
	valid, err := valueobjects.NewLocale(locale)
	if err != nil {
		return err
	}
	u.Locale = valid.String()
	return nil
}

// IsEmailVerified returns true if the user has verified their email address.
func (u *User) IsEmailVerified() bool {
	return u.EmailVerifiedAt != nil
}

// VerifyEmail marks the user's email as verified at the current time.
func (u *User) VerifyEmail() {
	now := time.Now()
	u.EmailVerifiedAt = &now
}
