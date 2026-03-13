package auth_test

import (
	"testing"

	"ductifact/internal/config"
	"ductifact/internal/infrastructure/auth"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testJWTConfig = config.JWT{
	Secret: "test-secret-key-at-least-32-chars!!",
}

// helper creates a JWTProvider with a test secret.
func newTestProvider(t *testing.T) *auth.JWTProvider {
	t.Helper()
	return auth.NewJWTProvider(testJWTConfig)
}

// =============================================================================
// NewJWTProvider
// =============================================================================

func TestNewJWTProvider_WithoutSecret_Panics(t *testing.T) {
	assert.Panics(t, func() {
		auth.NewJWTProvider(config.JWT{Secret: ""})
	})
}

func TestNewJWTProvider_WithSecret_DoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		auth.NewJWTProvider(config.JWT{Secret: "a-valid-secret-for-testing-purposes"})
	})
}

// =============================================================================
// GenerateToken
// =============================================================================

func TestGenerateToken_ReturnsNonEmptyString(t *testing.T) {
	provider := newTestProvider(t)

	token, err := provider.GenerateToken(uuid.New(), "juan@example.com")

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestGenerateToken_DifferentUsersGetDifferentTokens(t *testing.T) {
	provider := newTestProvider(t)

	token1, _ := provider.GenerateToken(uuid.New(), "juan@example.com")
	token2, _ := provider.GenerateToken(uuid.New(), "pedro@example.com")

	assert.NotEqual(t, token1, token2)
}

// =============================================================================
// ValidateToken
// =============================================================================

func TestValidateToken_WithValidToken_ReturnsClaims(t *testing.T) {
	provider := newTestProvider(t)
	userID := uuid.New()
	email := "juan@example.com"

	token, err := provider.GenerateToken(userID, email)
	require.NoError(t, err)

	claims, err := provider.ValidateToken(token)

	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
}

func TestValidateToken_WithInvalidSignature_ReturnsError(t *testing.T) {
	// Generate with one secret
	provider1 := auth.NewJWTProvider(config.JWT{Secret: "secret-key-one-at-least-32-chars!"})

	token, err := provider1.GenerateToken(uuid.New(), "juan@example.com")
	require.NoError(t, err)

	// Validate with a different secret
	provider2 := auth.NewJWTProvider(config.JWT{Secret: "secret-key-two-at-least-32-chars!"})

	claims, err := provider2.ValidateToken(token)

	assert.Nil(t, claims)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestValidateToken_WithGarbageString_ReturnsError(t *testing.T) {
	provider := newTestProvider(t)

	claims, err := provider.ValidateToken("this-is-not-a-jwt")

	assert.Nil(t, claims)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestValidateToken_WithEmptyString_ReturnsError(t *testing.T) {
	provider := newTestProvider(t)

	claims, err := provider.ValidateToken("")

	assert.Nil(t, claims)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestValidateToken_WithTamperedPayload_ReturnsError(t *testing.T) {
	provider := newTestProvider(t)

	token, err := provider.GenerateToken(uuid.New(), "juan@example.com")
	require.NoError(t, err)

	tampered := token[:len(token)-5] + "XXXXX"

	claims, err := provider.ValidateToken(tampered)

	assert.Nil(t, claims)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

// =============================================================================
// Round-trip: Generate → Validate
// =============================================================================

func TestRoundTrip_GenerateAndValidate_PreservesUserData(t *testing.T) {
	provider := newTestProvider(t)

	userID := uuid.New()
	email := "maria@example.com"

	token, err := provider.GenerateToken(userID, email)
	require.NoError(t, err)

	claims, err := provider.ValidateToken(token)
	require.NoError(t, err)

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
}
