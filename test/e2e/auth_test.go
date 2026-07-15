package e2e

import (
	"net/http"
	"testing"
	"time"

	"ductifact/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// seedOTP inserts a one-time OTP row with a known plaintext code and purpose,
// so the OTP-based endpoints can be exercised end-to-end.
func seedOTP(t *testing.T, email, purpose, code string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	require.NoError(t, err)
	err = env.db.Exec(
		"INSERT INTO one_time_otps (id, email, purpose, code_hash, expires_at, attempts, created_at) VALUES (gen_random_uuid(), ?, ?, ?, ?, 0, NOW())",
		email, purpose, string(hash), time.Now().Add(15*time.Minute),
	).Error
	require.NoError(t, err)
}

// seedRegistrationOTP seeds a registration OTP with a known plaintext code.
func seedRegistrationOTP(t *testing.T, email, code string) {
	t.Helper()
	seedOTP(t, email, "registration", code)
}

// ─── Start Registration ──────────────────────────────────────────────────────

func TestE2E_StartRegistration_Success(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"email": "juan@example.com",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// An OTP row should have been created for the email.
	var count int64
	err := env.db.Raw("SELECT COUNT(*) FROM one_time_otps WHERE purpose = 'registration' AND email = ?", "juan@example.com").Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestE2E_StartRegistration_InvalidEmail_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"email": "not-an-email",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_StartRegistration_EmptyBody_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Complete Registration (verify) ──────────────────────────────────────────

func TestE2E_Register_Success(t *testing.T) {
	clean(t)
	seedRegistrationOTP(t, "juan@example.com", "123456")

	resp := helpers.PostJSON(t, url("/auth/register/verify"), map[string]string{
		"email":    "juan@example.com",
		"code":     "123456",
		"name":     "Juan",
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

	// The OTP should have been consumed.
	var count int64
	require.NoError(t, env.db.Raw("SELECT COUNT(*) FROM one_time_otps WHERE purpose = 'registration' AND email = ?", "juan@example.com").Scan(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestE2E_Register_WrongCode_Returns400(t *testing.T) {
	clean(t)
	seedRegistrationOTP(t, "juan@example.com", "123456")

	resp := helpers.PostJSON(t, url("/auth/register/verify"), map[string]string{
		"email":    "juan@example.com",
		"code":     "000000",
		"name":     "Juan",
		"password": "securepass123",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "invalid or expired verification code")
}

func TestE2E_Register_NoOTP_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/register/verify"), map[string]string{
		"email":    "juan@example.com",
		"code":     "123456",
		"name":     "Juan",
		"password": "securepass123",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_Register_MissingName_Returns400(t *testing.T) {
	clean(t)
	seedRegistrationOTP(t, "juan@example.com", "123456")

	resp := helpers.PostJSON(t, url("/auth/register/verify"), map[string]string{
		"email":    "juan@example.com",
		"code":     "123456",
		"password": "securepass123",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_Register_ShortPassword_Returns400(t *testing.T) {
	clean(t)
	seedRegistrationOTP(t, "juan@example.com", "123456")

	resp := helpers.PostJSON(t, url("/auth/register/verify"), map[string]string{
		"email":    "juan@example.com",
		"code":     "123456",
		"name":     "Juan",
		"password": "short",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_Register_DuplicateEmail_Returns409(t *testing.T) {
	clean(t)

	// Pre-existing account.
	registerUser(t, "Juan", "same@example.com", "securepass123")

	// A stale OTP for the same email; completing it must fail with conflict.
	seedRegistrationOTP(t, "same@example.com", "123456")
	resp := helpers.PostJSON(t, url("/auth/register/verify"), map[string]string{
		"email":    "same@example.com",
		"code":     "123456",
		"name":     "Pedro",
		"password": "securepass123",
	})

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "email already in use")
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestE2E_Login_Success(t *testing.T) {
	clean(t)

	// Register first
	id, _ := registerUser(t, "Juan", "juan@example.com", "securepass123")

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
	assert.Equal(t, id, user["id"])
	assert.Equal(t, "Juan", user["name"])
	assert.Equal(t, "juan@example.com", user["email"])
}

func TestE2E_Login_WrongPassword_Returns401(t *testing.T) {
	clean(t)

	// Register
	registerUser(t, "Juan", "juan@example.com", "securepass123")

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
	_, token := registerUser(t, "Juan", "juan@example.com", "oldpass123")

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
	_, token := registerUser(t, "Juan", "juan@example.com", "oldpass123")

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
	_, token := registerUser(t, "Juan", "juan@example.com", "oldpass123")

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
	_, token := registerUser(t, "Juan", "juan@example.com", "oldpass123")

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

// ─── Forgot Password ────────────────────────────────────────────────────────

func TestE2E_ForgotPassword_WithExistingEmail_Returns200(t *testing.T) {
	clean(t)

	// Register a user
	registerUser(t, "Juan", "juan@example.com", "securepass123")

	// Request password reset
	resp := helpers.PostJSON(t, url("/auth/password/reset"), map[string]string{
		"email": "juan@example.com",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["message"], "password reset code")

	// Verify a password-reset OTP was created in DB
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM one_time_otps WHERE purpose = 'password_reset'").Scan(&count)
	assert.Equal(t, int64(1), count)
}

func TestE2E_ForgotPassword_WithNonExistingEmail_Returns200(t *testing.T) {
	clean(t)

	// Request password reset for non-existing email (should not reveal if email exists)
	resp := helpers.PostJSON(t, url("/auth/password/reset"), map[string]string{
		"email": "nonexistent@example.com",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["message"], "password reset code")
}

func TestE2E_ForgotPassword_MissingEmail_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/password/reset"), map[string]string{})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_ForgotPassword_InvalidEmailFormat_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/password/reset"), map[string]string{
		"email": "4k@5X0vM0X0jW.lEFo89.rapos.ZR--qNHX.7X.p.1XioK0IeNS.Uslb.OrWqNyvRbdngop",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_ForgotPassword_WithPendingOTP_DoesNotReplaceExistingCode(t *testing.T) {
	clean(t)

	registerUser(t, "Juan", "juan@example.com", "securepass123")
	seedOTP(t, "juan@example.com", "password_reset", "654321")

	resp := helpers.PostJSON(t, url("/auth/password/reset"), map[string]string{
		"email": "juan@example.com",
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Existing OTP should remain valid, proving it was not replaced.
	verifyResp := helpers.PostJSON(t, url("/auth/password/reset/verify"), map[string]string{
		"email":        "juan@example.com",
		"code":         "654321",
		"new_password": "newpass456",
	})
	assert.Equal(t, http.StatusOK, verifyResp.StatusCode)
}

// ─── Reset Password ─────────────────────────────────────────────────────────

func TestE2E_ResetPassword_WithValidCode_Returns200(t *testing.T) {
	clean(t)

	// Register a user
	registerUser(t, "Juan", "juan@example.com", "securepass123")

	// Seed a password-reset OTP with a known code
	seedOTP(t, "juan@example.com", "password_reset", "654321")

	// Reset the password
	resp := helpers.PostJSON(t, url("/auth/password/reset/verify"), map[string]string{
		"email":        "juan@example.com",
		"code":         "654321",
		"new_password": "newpass456",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["message"], "password reset successfully")

	// Verify login works with new password
	loginResp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"email":    "juan@example.com",
		"password": "newpass456",
	})
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)

	// Verify old password no longer works
	oldLoginResp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"email":    "juan@example.com",
		"password": "securepass123",
	})
	assert.Equal(t, http.StatusUnauthorized, oldLoginResp.StatusCode)
}

func TestE2E_ResetPassword_WithInvalidCode_Returns400(t *testing.T) {
	clean(t)

	registerUser(t, "Juan", "juan@example.com", "securepass123")
	seedOTP(t, "juan@example.com", "password_reset", "654321")

	resp := helpers.PostJSON(t, url("/auth/password/reset/verify"), map[string]string{
		"email":        "juan@example.com",
		"code":         "000000",
		"new_password": "newpass456",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "invalid or expired")
}

func TestE2E_ResetPassword_WithShortPassword_Returns400(t *testing.T) {
	clean(t)

	registerUser(t, "Juan", "juan@example.com", "securepass123")
	seedOTP(t, "juan@example.com", "password_reset", "654321")

	// Try to reset with a too-short password
	resp := helpers.PostJSON(t, url("/auth/password/reset/verify"), map[string]string{
		"email":        "juan@example.com",
		"code":         "654321",
		"new_password": "short",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_ResetPassword_MissingFields_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/auth/password/reset/verify"), map[string]string{
		"email": "juan@example.com",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
