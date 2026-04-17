package valueobjects

import (
	"crypto/sha256"
	"errors"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong  = errors.New("password must not exceed 72 characters")
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
//
// To avoid bcrypt's 72-byte limit on multi-byte Unicode passwords we
// pre-hash the input with SHA-256 (Dropbox pattern). The resulting
// 32-byte digest is always within the limit.
func NewPassword(raw string) (*Password, error) {
	if raw == "" {
		return nil, ErrPasswordEmpty
	}
	if utf8.RuneCountInString(raw) < 8 {
		return nil, ErrPasswordTooShort
	}
	if utf8.RuneCountInString(raw) > 72 {
		return nil, ErrPasswordTooLong
	}

	// SHA-256 pre-hash: produces a fixed 32-byte key, well within
	// bcrypt's 72-byte input limit regardless of the original encoding.
	preHash := sha256.Sum256([]byte(raw))

	// bcrypt.GenerateFromPassword:
	// - Adds a random salt automatically (each hash differs even for the same password)
	// - bcrypt.DefaultCost = 10 → 2^10 iterations. Higher = more secure but slower.
	hash, err := bcrypt.GenerateFromPassword(preHash[:], bcrypt.DefaultCost)
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
	preHash := sha256.Sum256([]byte(raw))
	err := bcrypt.CompareHashAndPassword([]byte(p.hash), preHash[:])
	if err != nil {
		return ErrInvalidPassword
	}
	return nil
}

// Hash returns the bcrypt hash string (for storage in DB).
func (p *Password) Hash() string {
	return p.hash
}
