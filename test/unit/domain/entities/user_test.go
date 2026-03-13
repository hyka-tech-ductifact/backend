package entities_test

import (
	"testing"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser_WithValidData_ReturnsUser(t *testing.T) {
	user, err := entities.NewUser("Juan", "juan@example.com", "securepass123")

	require.NoError(t, err)
	assert.Equal(t, "Juan", user.Name)
	assert.Equal(t, "juan@example.com", user.Email)
	assert.NotEmpty(t, user.ID)
	assert.NotEmpty(t, user.PasswordHash, "PasswordHash must not be empty")
	assert.NotEqual(t, "securepass123", user.PasswordHash, "PasswordHash must not be the raw password")
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
}

func TestNewUser_WithEmptyName_ReturnsError(t *testing.T) {
	user, err := entities.NewUser("", "juan@example.com", "securepass123")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, entities.ErrEmptyUserName)
}

func TestNewUser_WithInvalidEmail_ReturnsError(t *testing.T) {
	user, err := entities.NewUser("Juan", "invalid-email", "securepass123")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, valueobjects.ErrInvalidEmail)
}

func TestNewUser_WithEmptyEmail_ReturnsError(t *testing.T) {
	user, err := entities.NewUser("Juan", "", "securepass123")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, valueobjects.ErrInvalidEmail)
}

func TestNewUser_WithEmptyNameAndInvalidEmail_ReturnsNameError(t *testing.T) {
	// Name is validated first, so we get the name error
	user, err := entities.NewUser("", "invalid-email", "securepass123")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, entities.ErrEmptyUserName)
}

func TestNewUser_GeneratesUniqueIDs(t *testing.T) {
	user1, err1 := entities.NewUser("User1", "user1@example.com", "securepass123")
	user2, err2 := entities.NewUser("User2", "user2@example.com", "securepass123")

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, user1.ID, user2.ID, "each user must have a unique ID")
}

func TestNewUser_SetsCreatedAtAndUpdatedAtEqual(t *testing.T) {
	user, err := entities.NewUser("Juan", "juan@example.com", "securepass123")

	require.NoError(t, err)
	assert.Equal(t, user.CreatedAt, user.UpdatedAt,
		"CreatedAt and UpdatedAt must be equal on creation")
}

// --- Password validation tests ---

func TestNewUser_WithShortPassword_ReturnsError(t *testing.T) {
	user, err := entities.NewUser("Juan", "juan@example.com", "short")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, valueobjects.ErrPasswordTooShort)
}

func TestNewUser_WithEmptyPassword_ReturnsError(t *testing.T) {
	user, err := entities.NewUser("Juan", "juan@example.com", "")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, valueobjects.ErrPasswordEmpty)
}

func TestNewUser_HashCanBeVerified(t *testing.T) {
	rawPassword := "securepass123"
	user, err := entities.NewUser("Juan", "juan@example.com", rawPassword)

	require.NoError(t, err)

	// Reconstruct the Password VO from the stored hash and verify
	pwd := valueobjects.NewPasswordFromHash(user.PasswordHash)
	assert.NoError(t, pwd.Compare(rawPassword))
	assert.Error(t, pwd.Compare("wrongpassword"))
}
