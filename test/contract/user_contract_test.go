package contract_test

import (
	"net/http"
	"testing"

	"ductifact/test/helpers"
)

// ═══════════════════════════════════════════════════════════════
// GET /users/me
// ═══════════════════════════════════════════════════════════════

func TestContract_GetMe_Authenticated_Returns200_WithUserResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Me User", "contract-getme@example.com", "securepass123")

	resp := helpers.AuthGetJSON(t, url("/users/me"), token)

	cv.ValidateResponse(resp, http.StatusOK)
}

func TestContract_GetMe_NoToken_Returns401_WithErrorResponse(t *testing.T) {
	cv := newValidator(t)

	resp := helpers.GetJSON(t, url("/users/me"))

	cv.ValidateResponse(resp, http.StatusUnauthorized)
}

func TestContract_GetMe_InvalidToken_Returns401_WithErrorResponse(t *testing.T) {
	cv := newValidator(t)

	resp := helpers.AuthGetJSON(t, url("/users/me"), "invalid-jwt-token")

	cv.ValidateResponse(resp, http.StatusUnauthorized)
}

// ═══════════════════════════════════════════════════════════════
// PUT /users/me
// ═══════════════════════════════════════════════════════════════

func TestContract_UpdateMe_ValidBody_Returns200_WithUserResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Update User", "contract-update@example.com", "securepass123")

	resp := helpers.AuthPutJSON(t, url("/users/me"), token, map[string]string{
		"name": "Updated Name",
	})

	cv.ValidateResponse(resp, http.StatusOK)
}

func TestContract_UpdateMe_NoToken_Returns401_WithErrorResponse(t *testing.T) {
	cv := newValidator(t)

	resp := helpers.PutJSON(t, url("/users/me"), map[string]string{
		"name": "Ghost",
	})

	cv.ValidateResponse(resp, http.StatusUnauthorized)
}

func TestContract_UpdateMe_InvalidEmail_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Update User", "contract-update400@example.com", "securepass123")

	resp := helpers.AuthPutJSON(t, url("/users/me"), token, map[string]string{
		"email": "not-an-email",
	})

	cv.ValidateResponse(resp, http.StatusBadRequest)
}

func TestContract_UpdateMe_DuplicateEmail_Returns409_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	// Create two users
	registerAndLogin(t, "First User", "contract-first@example.com", "securepass123")
	token := registerAndLogin(t, "Second User", "contract-second@example.com", "securepass123")

	// Try to update second user's email to first user's email
	resp := helpers.AuthPutJSON(t, url("/users/me"), token, map[string]string{
		"email": "contract-first@example.com",
	})

	cv.ValidateResponse(resp, http.StatusConflict)
}
