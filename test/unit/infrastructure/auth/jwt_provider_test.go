package auth_test

import (
	"testing"
	"time"

	"ductifact/internal/config"
	"ductifact/internal/infrastructure/auth"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testJWTConfig = config.JWT{
	Secret:               "test-secret-key-at-least-32-chars!!",
	TokenDuration:        15 * time.Minute,
	RefreshTokenDuration: 7 * 24 * time.Hour,
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
// GenerateTokenPair
// =============================================================================

func TestGenerateTokenPair_ReturnsBothTokens(t *testing.T) {
	provider := newTestProvider(t)

	pair, err := provider.GenerateTokenPair(uuid.New(), "juan@example.com")

	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.NotEqual(t, pair.AccessToken, pair.RefreshToken)
}

func TestGenerateTokenPair_DifferentUsersGetDifferentTokens(t *testing.T) {
	provider := newTestProvider(t)

	pair1, _ := provider.GenerateTokenPair(uuid.New(), "juan@example.com")
	pair2, _ := provider.GenerateTokenPair(uuid.New(), "pedro@example.com")

	assert.NotEqual(t, pair1.AccessToken, pair2.AccessToken)
	assert.NotEqual(t, pair1.RefreshToken, pair2.RefreshToken)
}

// =============================================================================
// ValidateToken (access tokens only)
// =============================================================================

func TestValidateToken_WithValidAccessToken_ReturnsClaims(t *testing.T) {
	provider := newTestProvider(t)
	userID := uuid.New()
	email := "juan@example.com"

	pair, err := provider.GenerateTokenPair(userID, email)
	require.NoError(t, err)

	claims, err := provider.ValidateToken(pair.AccessToken)

	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
}

func TestValidateToken_WithRefreshToken_ReturnsError(t *testing.T) {
	provider := newTestProvider(t)

	pair, err := provider.GenerateTokenPair(uuid.New(), "juan@example.com")
	require.NoError(t, err)

	// A refresh token must NOT be accepted as an access token
	claims, err := provider.ValidateToken(pair.RefreshToken)

	assert.Nil(t, claims)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestValidateToken_WithInvalidSignature_ReturnsError(t *testing.T) {
	// Generate with one secret
	provider1 := auth.NewJWTProvider(config.JWT{
		Secret:               "secret-key-one-at-least-32-chars!",
		TokenDuration:        15 * time.Minute,
		RefreshTokenDuration: 168 * time.Hour,
	})

	pair, err := provider1.GenerateTokenPair(uuid.New(), "juan@example.com")
	require.NoError(t, err)

	// Validate with a different secret
	provider2 := auth.NewJWTProvider(config.JWT{
		Secret:               "secret-key-two-at-least-32-chars!",
		TokenDuration:        15 * time.Minute,
		RefreshTokenDuration: 168 * time.Hour,
	})

	claims, err := provider2.ValidateToken(pair.AccessToken)

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

	pair, err := provider.GenerateTokenPair(uuid.New(), "juan@example.com")
	require.NoError(t, err)

	tampered := pair.AccessToken[:len(pair.AccessToken)-5] + "XXXXX"

	claims, err := provider.ValidateToken(tampered)

	assert.Nil(t, claims)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

// =============================================================================
// ValidateRefreshToken
// =============================================================================

func TestValidateRefreshToken_WithValidRefreshToken_ReturnsClaims(t *testing.T) {
	provider := newTestProvider(t)
	userID := uuid.New()
	email := "juan@example.com"

	pair, err := provider.GenerateTokenPair(userID, email)
	require.NoError(t, err)

	claims, err := provider.ValidateRefreshToken(pair.RefreshToken)

	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
}

func TestValidateRefreshToken_WithAccessToken_ReturnsError(t *testing.T) {
	provider := newTestProvider(t)

	pair, err := provider.GenerateTokenPair(uuid.New(), "juan@example.com")
	require.NoError(t, err)

	// An access token must NOT be accepted as a refresh token
	claims, err := provider.ValidateRefreshToken(pair.AccessToken)

	assert.Nil(t, claims)
	assert.ErrorIs(t, err, auth.ErrInvalidRefreshToken)
}

func TestValidateRefreshToken_WithGarbageString_ReturnsError(t *testing.T) {
	provider := newTestProvider(t)

	claims, err := provider.ValidateRefreshToken("this-is-not-a-jwt")

	assert.Nil(t, claims)
	assert.ErrorIs(t, err, auth.ErrInvalidRefreshToken)
}

// =============================================================================
// Round-trip: GenerateTokenPair → Validate
// =============================================================================

func TestRoundTrip_GenerateAndValidate_PreservesUserData(t *testing.T) {
	provider := newTestProvider(t)

	userID := uuid.New()
	email := "maria@example.com"

	pair, err := provider.GenerateTokenPair(userID, email)
	require.NoError(t, err)

	// Access token round-trip
	accessClaims, err := provider.ValidateToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, accessClaims.UserID)
	assert.Equal(t, email, accessClaims.Email)

	// Refresh token round-trip
	refreshClaims, err := provider.ValidateRefreshToken(pair.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, userID, refreshClaims.UserID)
	assert.Equal(t, email, refreshClaims.Email)
}
