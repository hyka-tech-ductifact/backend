package contract_test

import (
	"net/http"
	"testing"

	"ductifact/test/helpers"
)

// ═══════════════════════════════════════════════════════════════
// POST /auth/register
// ═══════════════════════════════════════════════════════════════

func TestContract_Register_ValidBody_Returns201_WithAuthResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Contract User",
		"email":    "contract-register@example.com",
		"password": "securepass123",
	})

	// Validates against AuthResponse schema (including nested UserResponse)
	cv.ValidateResponse(resp, http.StatusCreated)
}

func TestContract_Register_MissingName_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"email":    "contract-noname@example.com",
		"password": "securepass123",
	})

	cv.ValidateResponse(resp, http.StatusBadRequest)
}

func TestContract_Register_MissingEmail_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "No Email",
		"password": "securepass123",
	})

	cv.ValidateResponse(resp, http.StatusBadRequest)
}

func TestContract_Register_MissingPassword_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":  "No Password",
		"email": "contract-nopw@example.com",
	})

	cv.ValidateResponse(resp, http.StatusBadRequest)
}

func TestContract_Register_InvalidEmail_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Bad Email",
		"email":    "not-an-email",
		"password": "securepass123",
	})

	cv.ValidateResponse(resp, http.StatusBadRequest)
}

func TestContract_Register_ShortPassword_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Short PW",
		"email":    "contract-shortpw@example.com",
		"password": "short",
	})

	cv.ValidateResponse(resp, http.StatusBadRequest)
}

func TestContract_Register_DuplicateEmail_Returns409_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	// First registration — just to seed data
	resp1 := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "First User",
		"email":    "contract-dup@example.com",
		"password": "securepass123",
	})
	defer resp1.Body.Close()

	// Second registration with the same email
	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Second User",
		"email":    "contract-dup@example.com",
		"password": "securepass123",
	})

	cv.ValidateResponse(resp, http.StatusConflict)
}

func TestContract_Register_EmptyBody_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{})

	cv.ValidateResponse(resp, http.StatusBadRequest)
}

// ═══════════════════════════════════════════════════════════════
// POST /auth/login
// ═══════════════════════════════════════════════════════════════

func TestContract_Login_ValidCredentials_Returns200_WithAuthResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	// Seed a user
	reg := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Login User",
		"email":    "contract-login@example.com",
		"password": "securepass123",
	})
	defer reg.Body.Close()

	// Login
	resp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"email":    "contract-login@example.com",
		"password": "securepass123",
	})

	cv.ValidateResponse(resp, http.StatusOK)
}

func TestContract_Login_WrongPassword_Returns401_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	// Seed
	reg := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     "Login User",
		"email":    "contract-loginwrong@example.com",
		"password": "securepass123",
	})
	defer reg.Body.Close()

	resp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"email":    "contract-loginwrong@example.com",
		"password": "wrongpassword",
	})

	cv.ValidateResponse(resp, http.StatusUnauthorized)
}

func TestContract_Login_NonExistentEmail_Returns401_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	resp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"email":    "contract-ghost@example.com",
		"password": "securepass123",
	})

	cv.ValidateResponse(resp, http.StatusUnauthorized)
}

func TestContract_Login_MissingEmail_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	resp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"password": "securepass123",
	})

	cv.ValidateResponse(resp, http.StatusBadRequest)
}

func TestContract_Login_MissingPassword_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	resp := helpers.PostJSON(t, url("/auth/login"), map[string]string{
		"email": "contract-nopw@example.com",
	})

	cv.ValidateResponse(resp, http.StatusBadRequest)
}

func TestContract_Login_EmptyBody_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	resp := helpers.PostJSON(t, url("/auth/login"), map[string]string{})

	cv.ValidateResponse(resp, http.StatusBadRequest)
}
