package valueobjects

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordEmpty    = errors.New("password cannot be empty")
	ErrInvalidPassword  = errors.New("invalid password")
)

// Password is a value object that handles password validation and hashing.
// It stores the bcrypt hash, never the raw password.
type Password struct {
	hash string
}

// NewPassword validates the raw password and returns a Password with the bcrypt hash.
// The raw password is never stored.
func NewPassword(raw string) (*Password, error) {
	if raw == "" {
		return nil, ErrPasswordEmpty
	}
	if len(raw) < 8 {
		return nil, ErrPasswordTooShort
	}

	// bcrypt.GenerateFromPassword:
	// - Adds a random salt automatically (each hash differs even for the same password)
	// - bcrypt.DefaultCost = 10 → 2^10 iterations. Higher = more secure but slower.
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return &Password{hash: string(hash)}, nil
}

// NewPasswordFromHash creates a Password from an already-hashed value.
// Used when loading from the database (the hash is already computed).
func NewPasswordFromHash(hash string) *Password {
	return &Password{hash: hash}
}

// Compare checks if the given raw password matches the stored hash.
// Returns nil on success, ErrInvalidPassword on failure.
func (p *Password) Compare(raw string) error {
	err := bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(raw))
	if err != nil {
		return ErrInvalidPassword
	}
	return nil
}

// Hash returns the bcrypt hash string (for storage in DB).
func (p *Password) Hash() string {
	return p.hash
}
