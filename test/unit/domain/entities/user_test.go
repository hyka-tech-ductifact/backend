package entities_test

import (
	"testing"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validUserParams returns a CreateUserParams with sensible defaults for tests.
func validUserParams() entities.CreateUserParams {
	return entities.CreateUserParams{
		Name:     "Juan",
		Email:    "juan@example.com",
		Password: "securepass123",
		Locale:   "en",
	}
}

func TestNewUser_WithValidData_ReturnsUser(t *testing.T) {
	user, err := entities.NewUser(validUserParams())

	require.NoError(t, err)
	assert.Equal(t, "Juan", user.Name)
	assert.Equal(t, "juan@example.com", user.Email)
	assert.Equal(t, "en", user.Locale)
	assert.NotEmpty(t, user.ID)
	assert.NotEmpty(t, user.PasswordHash, "PasswordHash must not be empty")
	assert.NotEqual(t, "securepass123", user.PasswordHash, "PasswordHash must not be the raw password")
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
}

func TestNewUser_WithEmptyName_ReturnsError(t *testing.T) {
	p := validUserParams()
	p.Name = ""

	user, err := entities.NewUser(p)

	assert.Nil(t, user)
	assert.ErrorIs(t, err, entities.ErrEmptyUserName)
}

func TestNewUser_WithInvalidEmail_ReturnsError(t *testing.T) {
	p := validUserParams()
	p.Email = "invalid-email"

	user, err := entities.NewUser(p)

	assert.Nil(t, user)
	assert.ErrorIs(t, err, valueobjects.ErrInvalidEmail)
}

func TestNewUser_WithEmptyEmail_ReturnsError(t *testing.T) {
	p := validUserParams()
	p.Email = ""

	user, err := entities.NewUser(p)

	assert.Nil(t, user)
	assert.ErrorIs(t, err, valueobjects.ErrInvalidEmail)
}

func TestNewUser_WithEmptyNameAndInvalidEmail_ReturnsNameError(t *testing.T) {
	// Name is validated first, so we get the name error
	p := validUserParams()
	p.Name = ""
	p.Email = "invalid-email"

	user, err := entities.NewUser(p)

	assert.Nil(t, user)
	assert.ErrorIs(t, err, entities.ErrEmptyUserName)
}

func TestNewUser_GeneratesUniqueIDs(t *testing.T) {
	p1 := validUserParams()
	p1.Name = "User1"
	p1.Email = "user1@example.com"

	p2 := validUserParams()
	p2.Name = "User2"
	p2.Email = "user2@example.com"

	user1, err1 := entities.NewUser(p1)
	user2, err2 := entities.NewUser(p2)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, user1.ID, user2.ID, "each user must have a unique ID")
}

func TestNewUser_SetsCreatedAtAndUpdatedAtEqual(t *testing.T) {
	user, err := entities.NewUser(validUserParams())

	require.NoError(t, err)
	assert.Equal(t, user.CreatedAt, user.UpdatedAt,
		"CreatedAt and UpdatedAt must be equal on creation")
}

// --- Password validation tests ---

func TestNewUser_WithShortPassword_ReturnsError(t *testing.T) {
	p := validUserParams()
	p.Password = "short"

	user, err := entities.NewUser(p)

	assert.Nil(t, user)
	assert.ErrorIs(t, err, valueobjects.ErrPasswordTooShort)
}

func TestNewUser_WithEmptyPassword_ReturnsError(t *testing.T) {
	p := validUserParams()
	p.Password = ""

	user, err := entities.NewUser(p)

	assert.Nil(t, user)
	assert.ErrorIs(t, err, valueobjects.ErrPasswordEmpty)
}

func TestNewUser_HashCanBeVerified(t *testing.T) {
	p := validUserParams()

	user, err := entities.NewUser(p)

	require.NoError(t, err)

	// Reconstruct the Password VO from the stored hash and verify
	pwd := valueobjects.NewPasswordFromHash(user.PasswordHash)
	assert.NoError(t, pwd.Compare(p.Password))
	assert.Error(t, pwd.Compare("wrongpassword"))
}

// --- NewUser locale tests ---

func TestNewUser_WithSpanishLocale_SetsLocale(t *testing.T) {
	p := validUserParams()
	p.Locale = "es"

	user, err := entities.NewUser(p)

	require.NoError(t, err)
	assert.Equal(t, "es", user.Locale)
}

func TestNewUser_WithEmptyLocale_ReturnsError(t *testing.T) {
	p := validUserParams()
	p.Locale = ""

	user, err := entities.NewUser(p)

	assert.Nil(t, user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid locale")
}

func TestNewUser_WithInvalidLocale_ReturnsError(t *testing.T) {
	p := validUserParams()
	p.Locale = "fr"

	user, err := entities.NewUser(p)

	assert.Nil(t, user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid locale")
}

// --- SetLocale tests ---

func TestSetLocale_WithSupportedLocale_Updates(t *testing.T) {
	user, _ := entities.NewUser(validUserParams())

	err := user.SetLocale("es")

	require.NoError(t, err)
	assert.Equal(t, "es", user.Locale)
}

func TestSetLocale_WithEnglish_Updates(t *testing.T) {
	user, _ := entities.NewUser(validUserParams())
	_ = user.SetLocale("es") // change first

	err := user.SetLocale("en")

	require.NoError(t, err)
	assert.Equal(t, "en", user.Locale)
}

func TestSetLocale_WithUnsupportedLocale_ReturnsError(t *testing.T) {
	user, _ := entities.NewUser(validUserParams())

	err := user.SetLocale("fr")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid locale")
	assert.Equal(t, "en", user.Locale, "locale must not change on error")
}

// =============================================================================
// Email Verification
// =============================================================================

func TestIsEmailVerified_NewUser_ReturnsFalse(t *testing.T) {
	user, err := entities.NewUser(validUserParams())
	require.NoError(t, err)

	assert.False(t, user.IsEmailVerified())
	assert.Nil(t, user.EmailVerifiedAt)
}

func TestVerifyEmail_SetsEmailVerifiedAt(t *testing.T) {
	user, err := entities.NewUser(validUserParams())
	require.NoError(t, err)

	user.VerifyEmail()

	assert.True(t, user.IsEmailVerified())
	assert.NotNil(t, user.EmailVerifiedAt)
}

func TestVerifyEmail_DoesNotMutateUpdatedAt(t *testing.T) {
	user, err := entities.NewUser(validUserParams())
	require.NoError(t, err)
	originalUpdatedAt := user.UpdatedAt

	user.VerifyEmail()

	assert.Equal(t, originalUpdatedAt, user.UpdatedAt, "UpdatedAt is the repository's responsibility")
}
