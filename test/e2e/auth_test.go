package e2e

import (
	"net/http"
	"testing"

	"ductifact/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Register ────────────────────────────────────────────────────────────────

func TestE2E_Register_Success(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "juan@example.com",
		"password": "securepass123",
	})

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.NotEmpty(t, body["access_token"])
	assert.NotEmpty(t, body["refresh_token"])
	user := body["user"].(map[string]any)
	assert.NotEmpty(t, user["id"])
	assert.Equal(t, "Juan", user["name"])
	assert.Equal(t, "juan@example.com", user["email"])
}

func TestE2E_Register_MissingName_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"email":    "juan@example.com",
		"password": "securepass123",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_Register_InvalidEmail_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "not-an-email",
		"password": "securepass123",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_Register_ShortPassword_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "juan@example.com",
		"password": "short",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_Register_DuplicateEmail_Returns409(t *testing.T) {
	clean(t)

	// First registration
	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "same@example.com",
		"password": "securepass123",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Second registration with same email
	resp = helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Pedro",
		"email":    "same@example.com",
		"password": "securepass123",
	})

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "email already in use")
}

func TestE2E_Register_EmptyBody_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestE2E_Login_Success(t *testing.T) {
	clean(t)

	// Register first
	regResp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "juan@example.com",
		"password": "securepass123",
	})
	require.Equal(t, http.StatusCreated, regResp.StatusCode)
	regBody := helpers.ParseBody(t, regResp)
	regUser := regBody["user"].(map[string]any)

	// Login
	resp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"email":    "juan@example.com",
		"password": "securepass123",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.NotEmpty(t, body["access_token"])
	assert.NotEmpty(t, body["refresh_token"])
	user := body["user"].(map[string]any)
	assert.Equal(t, regUser["id"], user["id"])
	assert.Equal(t, "Juan", user["name"])
	assert.Equal(t, "juan@example.com", user["email"])
}

func TestE2E_Login_WrongPassword_Returns401(t *testing.T) {
	clean(t)

	// Register
	regResp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "juan@example.com",
		"password": "securepass123",
	})
	require.Equal(t, http.StatusCreated, regResp.StatusCode)
	regResp.Body.Close()

	// Login with wrong password
	resp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"email":    "juan@example.com",
		"password": "wrongpassword",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "invalid email or password")
}

func TestE2E_Login_NonExistentEmail_Returns401(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"email":    "noexiste@example.com",
		"password": "securepass123",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	// Same generic error — doesn't reveal if email exists
	assert.Contains(t, body["error"], "invalid email or password")
}

func TestE2E_Login_MissingEmail_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"password": "securepass123",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_Login_MissingPassword_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"email": "juan@example.com",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_Login_EmptyBody_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/login"), map[string]string{})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Change Password ─────────────────────────────────────────────────────────

func TestE2E_ChangePassword_Success(t *testing.T) {
	clean(t)

	// Register a user
	regResp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "juan@example.com",
		"password": "oldpass123",
	})
	require.Equal(t, http.StatusCreated, regResp.StatusCode)
	regBody := helpers.ParseBody(t, regResp)
	token := regBody["access_token"].(string)

	// Change password
	resp := helpers.AuthPutJSON(t, url("/auth/password"), token, map[string]string{
		"current_password": "oldpass123",
		"new_password":     "newpass456",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Equal(t, "password changed successfully", body["message"])

	// Verify the new password works by logging in
	loginResp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"email":    "juan@example.com",
		"password": "newpass456",
	})
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)
}

func TestE2E_ChangePassword_WrongCurrentPassword_Returns401(t *testing.T) {
	clean(t)

	// Register a user
	regResp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "juan@example.com",
		"password": "oldpass123",
	})
	require.Equal(t, http.StatusCreated, regResp.StatusCode)
	regBody := helpers.ParseBody(t, regResp)
	token := regBody["access_token"].(string)

	// Try to change with wrong current password
	resp := helpers.AuthPutJSON(t, url("/auth/password"), token, map[string]string{
		"current_password": "wrongpassword",
		"new_password":     "newpass456",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "current password is incorrect")
}

func TestE2E_ChangePassword_ShortNewPassword_Returns400(t *testing.T) {
	clean(t)

	// Register a user
	regResp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "juan@example.com",
		"password": "oldpass123",
	})
	require.Equal(t, http.StatusCreated, regResp.StatusCode)
	regBody := helpers.ParseBody(t, regResp)
	token := regBody["access_token"].(string)

	// Try to change with a too-short new password
	resp := helpers.AuthPutJSON(t, url("/auth/password"), token, map[string]string{
		"current_password": "oldpass123",
		"new_password":     "short",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_ChangePassword_MissingFields_Returns400(t *testing.T) {
	clean(t)

	// Register a user
	regResp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Juan",
		"email":    "juan@example.com",
		"password": "oldpass123",
	})
	require.Equal(t, http.StatusCreated, regResp.StatusCode)
	regBody := helpers.ParseBody(t, regResp)
	token := regBody["access_token"].(string)

	// Missing current_password
	resp := helpers.AuthPutJSON(t, url("/auth/password"), token, map[string]string{
		"new_password": "newpass456",
	})
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_ChangePassword_NoAuth_Returns401(t *testing.T) {
	clean(t)

	// Try without a token
	resp := helpers.PutJSON(t, url("/auth/password"), map[string]string{
		"current_password": "oldpass123",
		"new_password":     "newpass456",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
