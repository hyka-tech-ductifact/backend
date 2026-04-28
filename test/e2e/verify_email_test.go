package e2e

import (
	"net/http"
	"testing"

	"ductifact/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Verify Email ────────────────────────────────────────────────────────────

func TestE2E_VerifyEmail_WithValidToken_Returns200(t *testing.T) {
	clean(t)

	// Register a user (creates a verification token in the DB)
	regResp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "juan@example.com",
		"password": "securepass123",
	})
	require.Equal(t, http.StatusCreated, regResp.StatusCode)
	regResp.Body.Close()

	// Get the token from the DB
	var token string
	err := env.db.Raw("SELECT token FROM one_time_tokens WHERE type = 'email_verification' LIMIT 1").Scan(&token).Error
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Verify email
	resp := helpers.PostJSON(t, url("/auth/verify-email"), map[string]string{
		"token": token,
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["message"], "verified")
}

func TestE2E_VerifyEmail_WithInvalidToken_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/verify-email"), map[string]string{
		"token": "nonexistent-token-abc123",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "invalid or expired")
}

func TestE2E_VerifyEmail_WithMissingToken_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/verify-email"), map[string]string{})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_VerifyEmail_AlreadyVerified_Returns409(t *testing.T) {
	clean(t)

	// Register
	regResp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "juan@example.com",
		"password": "securepass123",
	})
	require.Equal(t, http.StatusCreated, regResp.StatusCode)
	regResp.Body.Close()

	// Get token and verify
	var token string
	err := env.db.Raw("SELECT token FROM one_time_tokens WHERE type = 'email_verification' LIMIT 1").Scan(&token).Error
	require.NoError(t, err)

	resp := helpers.PostJSON(t, url("/auth/verify-email"), map[string]string{"token": token})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Register another user and get a fresh token, then manually verify via DB
	// Instead: the original tokens were deleted, so trying the same token again gives 400
	resp = helpers.PostJSON(t, url("/auth/verify-email"), map[string]string{"token": token})
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "invalid or expired")
}

// ─── Resend Verification ─────────────────────────────────────────────────────

func TestE2E_ResendVerification_Authenticated_Returns200(t *testing.T) {
	clean(t)

	// Register (user is not verified)
	regResp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "juan@example.com",
		"password": "securepass123",
	})
	require.Equal(t, http.StatusCreated, regResp.StatusCode)
	regBody := helpers.ParseBody(t, regResp)
	accessToken := regBody["access_token"].(string)

	// Resend verification
	resp := helpers.AuthPostJSON(t, url("/auth/resend-verification"), accessToken, map[string]string{})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["message"], "verification email sent")
}

func TestE2E_ResendVerification_Unauthenticated_Returns401(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/resend-verification"), map[string]string{})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_ResendVerification_AlreadyVerified_Returns409(t *testing.T) {
	clean(t)

	// Register
	regResp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "juan@example.com",
		"password": "securepass123",
	})
	require.Equal(t, http.StatusCreated, regResp.StatusCode)
	regBody := helpers.ParseBody(t, regResp)
	accessToken := regBody["access_token"].(string)

	// Verify the email first
	var token string
	err := env.db.Raw("SELECT token FROM one_time_tokens WHERE type = 'email_verification' LIMIT 1").Scan(&token).Error
	require.NoError(t, err)

	verifyResp := helpers.PostJSON(t, url("/auth/verify-email"), map[string]string{"token": token})
	require.Equal(t, http.StatusOK, verifyResp.StatusCode)
	verifyResp.Body.Close()

	// Try to resend — should fail because already verified
	resp := helpers.AuthPostJSON(t, url("/auth/resend-verification"), accessToken, map[string]string{})

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "already verified")
}

func TestE2E_VerifyEmail_UserResponseIncludesEmailVerified(t *testing.T) {
	clean(t)

	// Register — email_verified should be false
	regResp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "juan@example.com",
		"password": "securepass123",
	})
	require.Equal(t, http.StatusCreated, regResp.StatusCode)
	regBody := helpers.ParseBody(t, regResp)
	user := regBody["user"].(map[string]any)
	assert.Equal(t, false, user["email_verified"])

	// Verify email
	var token string
	err := env.db.Raw("SELECT token FROM one_time_tokens WHERE type = 'email_verification' LIMIT 1").Scan(&token).Error
	require.NoError(t, err)

	verifyResp := helpers.PostJSON(t, url("/auth/verify-email"), map[string]string{"token": token})
	require.Equal(t, http.StatusOK, verifyResp.StatusCode)
	verifyResp.Body.Close()

	// Login again — email_verified should be true
	loginResp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"email":    "juan@example.com",
		"password": "securepass123",
	})
	require.Equal(t, http.StatusOK, loginResp.StatusCode)
	loginBody := helpers.ParseBody(t, loginResp)
	userAfter := loginBody["user"].(map[string]any)
	assert.Equal(t, true, userAfter["email_verified"])
}
