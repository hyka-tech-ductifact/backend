package contract_test

import (
	"net/http"
	"testing"

	"ductifact/test/helpers"
)

// ═══════════════════════════════════════════════════════════════
// POST /users/me/clients
// ═══════════════════════════════════════════════════════════════

func TestContract_CreateClient_ValidBody_Returns201_WithClientResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Client Owner", "contract-client-create@example.com", "securepass123").AccessToken

	resp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{
		"name":        "Acme Corp",
		"phone":       "+34 600 111 222",
		"email":       "acme@example.com",
		"description": "A test client",
	})

	cv.ValidateResponse(resp, http.StatusCreated)
}

func TestContract_CreateClient_MinimalBody_Returns201_WithClientResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Client Owner", "contract-client-create-min@example.com", "securepass123").AccessToken

	// Only required field: name. Optional fields default to "".
	resp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{
		"name": "Minimal Client",
	})

	cv.ValidateResponse(resp, http.StatusCreated)
}

func TestContract_CreateClient_MissingName_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Client Owner", "contract-client-noname@example.com", "securepass123").AccessToken

	resp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{})

	cv.ValidateResponse(resp, http.StatusBadRequest)
}

func TestContract_CreateClient_NoToken_Returns401_WithErrorResponse(t *testing.T) {
	cv := newValidator(t)

	resp := helpers.PostJSON(t, url("/users/me/clients"), map[string]string{
		"name": "Unauthorized Client",
	})

	cv.ValidateResponse(resp, http.StatusUnauthorized)
}

// ═══════════════════════════════════════════════════════════════
// GET /users/me/clients
// ═══════════════════════════════════════════════════════════════

func TestContract_ListClients_Empty_Returns200_WithEmptyArray(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "List Owner", "contract-client-list@example.com", "securepass123").AccessToken

	resp := helpers.AuthGetJSON(t, url("/users/me/clients"), token)

	cv.ValidateResponse(resp, http.StatusOK)
}

func TestContract_ListClients_WithItems_Returns200_WithClientResponseArray(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "List Owner", "contract-client-listitems@example.com", "securepass123").AccessToken

	// Create two clients with all fields
	r1 := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{
		"name":        "Client A",
		"phone":       "+34 600 111 222",
		"email":       "a@example.com",
		"description": "Client A desc",
	})
	defer r1.Body.Close()
	r2 := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{
		"name":        "Client B",
		"phone":       "+34 600 333 444",
		"email":       "b@example.com",
		"description": "Client B desc",
	})
	defer r2.Body.Close()

	resp := helpers.AuthGetJSON(t, url("/users/me/clients"), token)

	// Validates the array response — each item is checked against ClientResponse schema
	cv.ValidateResponse(resp, http.StatusOK)
}

func TestContract_ListClients_NoToken_Returns401_WithErrorResponse(t *testing.T) {
	cv := newValidator(t)

	resp := helpers.GetJSON(t, url("/users/me/clients"))

	cv.ValidateResponse(resp, http.StatusUnauthorized)
}

// ═══════════════════════════════════════════════════════════════
// GET /users/me/clients/:client_id
// ═══════════════════════════════════════════════════════════════

func TestContract_GetClient_Existing_Returns200_WithClientResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Get Owner", "contract-client-get@example.com", "securepass123").AccessToken

	// Create a client to GET (with all fields)
	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{
		"name":        "Get Client",
		"phone":       "+34 600 555 666",
		"email":       "getclient@example.com",
		"description": "Client to get",
	})
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)

	resp := helpers.AuthGetJSON(t, url("/users/me/clients/"+clientID), token)

	cv.ValidateResponse(resp, http.StatusOK)
}

func TestContract_GetClient_WrongUser_Returns403_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	// User A creates a client
	tokenA := registerAndLogin(t, "Owner A", "contract-get-owner-a@example.com", "securepass123").AccessToken
	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), tokenA, map[string]string{"name": "A's Client"})
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)

	// User B tries to read it
	tokenB := registerAndLogin(t, "Other B", "contract-get-other-b@example.com", "securepass123").AccessToken
	resp := helpers.AuthGetJSON(t, url("/users/me/clients/"+clientID), tokenB)

	cv.ValidateResponse(resp, http.StatusForbidden)
}

func TestContract_GetClient_InvalidUUID_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Get Owner", "contract-client-baduuid@example.com", "securepass123").AccessToken

	resp := helpers.AuthGetJSON(t, url("/users/me/clients/not-a-uuid"), token)

	cv.ValidateResponse(resp, http.StatusBadRequest)
}

func TestContract_GetClient_NotFound_Returns404_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Get Owner", "contract-client-notfound@example.com", "securepass123").AccessToken

	resp := helpers.AuthGetJSON(t, url("/users/me/clients/550e8400-e29b-41d4-a716-446655440000"), token)

	cv.ValidateResponse(resp, http.StatusNotFound)
}

func TestContract_GetClient_NoToken_Returns401_WithErrorResponse(t *testing.T) {
	cv := newValidator(t)

	resp := helpers.GetJSON(t, url("/users/me/clients/550e8400-e29b-41d4-a716-446655440000"))

	cv.ValidateResponse(resp, http.StatusUnauthorized)
}

// ═══════════════════════════════════════════════════════════════
// PUT /users/me/clients/:client_id
// ═══════════════════════════════════════════════════════════════

func TestContract_UpdateClient_ValidBody_Returns200_WithClientResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Update Owner", "contract-client-update@example.com", "securepass123").AccessToken

	// Create a client to update
	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{"name": "Original"})
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)

	resp := helpers.AuthPutJSON(t, url("/users/me/clients/"+clientID), token, map[string]string{
		"name":        "Updated Name",
		"phone":       "+34 600 777 888",
		"email":       "updated@example.com",
		"description": "Updated description",
	})

	cv.ValidateResponse(resp, http.StatusOK)
}

func TestContract_UpdateClient_PartialUpdate_Returns200_WithClientResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Partial Owner", "contract-client-partial@example.com", "securepass123").AccessToken

	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{
		"name":  "Original",
		"phone": "+34 600 111 222",
	})
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)

	// Only update phone — other fields should remain unchanged
	resp := helpers.AuthPutJSON(t, url("/users/me/clients/"+clientID), token, map[string]string{
		"phone": "+34 600 999 000",
	})

	cv.ValidateResponse(resp, http.StatusOK)
}

func TestContract_UpdateClient_WrongUser_Returns403_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	// User A creates a client
	tokenA := registerAndLogin(t, "Owner A", "contract-up-owner-a@example.com", "securepass123").AccessToken
	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), tokenA, map[string]string{"name": "A's Client"})
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)

	// User B tries to update it
	tokenB := registerAndLogin(t, "Other B", "contract-up-other-b@example.com", "securepass123").AccessToken
	resp := helpers.AuthPutJSON(t, url("/users/me/clients/"+clientID), tokenB, map[string]string{
		"name": "Hijacked",
	})

	cv.ValidateResponse(resp, http.StatusForbidden)
}

func TestContract_UpdateClient_NotFound_Returns404_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Update Owner", "contract-client-upnotfound@example.com", "securepass123").AccessToken

	resp := helpers.AuthPutJSON(t, url("/users/me/clients/550e8400-e29b-41d4-a716-446655440000"), token, map[string]string{
		"name": "Ghost",
	})

	cv.ValidateResponse(resp, http.StatusNotFound)
}

func TestContract_UpdateClient_InvalidUUID_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Update Owner", "contract-client-upbaduuid@example.com", "securepass123").AccessToken

	resp := helpers.AuthPutJSON(t, url("/users/me/clients/not-a-uuid"), token, map[string]string{
		"name": "Bad UUID",
	})

	cv.ValidateResponse(resp, http.StatusBadRequest)
}

func TestContract_UpdateClient_NoToken_Returns401_WithErrorResponse(t *testing.T) {
	cv := newValidator(t)

	resp := helpers.PutJSON(t, url("/users/me/clients/550e8400-e29b-41d4-a716-446655440000"), map[string]string{
		"name": "Unauthorized",
	})

	cv.ValidateResponse(resp, http.StatusUnauthorized)
}

// ═══════════════════════════════════════════════════════════════
// DELETE /users/me/clients/:client_id
// ═══════════════════════════════════════════════════════════════

func TestContract_DeleteClient_Existing_Returns204_NoBody(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Delete Owner", "contract-client-delete@example.com", "securepass123").AccessToken

	// Create a client to delete
	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{"name": "To Delete"})
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)

	resp := helpers.AuthDeleteJSON(t, url("/users/me/clients/"+clientID), token)

	cv.ValidateResponse(resp, http.StatusNoContent)
}

func TestContract_DeleteClient_WrongUser_Returns403_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	// User A creates a client
	tokenA := registerAndLogin(t, "Owner A", "contract-del-owner-a@example.com", "securepass123").AccessToken
	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), tokenA, map[string]string{"name": "A's Client"})
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)

	// User B tries to delete it
	tokenB := registerAndLogin(t, "Other B", "contract-del-other-b@example.com", "securepass123").AccessToken
	resp := helpers.AuthDeleteJSON(t, url("/users/me/clients/"+clientID), tokenB)

	cv.ValidateResponse(resp, http.StatusForbidden)
}

func TestContract_DeleteClient_NotFound_Returns404_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Delete Owner", "contract-client-delnotfound@example.com", "securepass123").AccessToken

	resp := helpers.AuthDeleteJSON(t, url("/users/me/clients/550e8400-e29b-41d4-a716-446655440000"), token)

	cv.ValidateResponse(resp, http.StatusNotFound)
}

func TestContract_DeleteClient_InvalidUUID_Returns400_WithErrorResponse(t *testing.T) {
	clean(t)
	cv := newValidator(t)

	token := registerAndLogin(t, "Delete Owner", "contract-client-delbaduuid@example.com", "securepass123").AccessToken

	resp := helpers.AuthDeleteJSON(t, url("/users/me/clients/not-a-uuid"), token)

	cv.ValidateResponse(resp, http.StatusBadRequest)
}

func TestContract_DeleteClient_NoToken_Returns401_WithErrorResponse(t *testing.T) {
	cv := newValidator(t)

	resp := helpers.DeleteJSON(t, url("/users/me/clients/550e8400-e29b-41d4-a716-446655440000"))

	cv.ValidateResponse(resp, http.StatusUnauthorized)
}
