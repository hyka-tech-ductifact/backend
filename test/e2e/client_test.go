package e2e

import (
	"net/http"
	"testing"

	"ductifact/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Helper: register a user via /auth/register and return (id, token) ---

func createUserForClients(t *testing.T, name, email string) (string, string) {
	t.Helper()
	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     name,
		"email":    email,
		"password": "securepass123",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	user := body["user"].(map[string]any)
	return user["id"].(string), body["access_token"].(string)
}

// ─── Create Client ───────────────────────────────────────────────────────────

func TestE2E_CreateClient_Success(t *testing.T) {
	clean(t)
	userID, token := createUserForClients(t, "Juan", "juan@example.com")

	resp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{
		"name": "Acme Corp",
	})

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.NotEmpty(t, body["id"])
	assert.Equal(t, "Acme Corp", body["name"])
	assert.Equal(t, userID, body["user_id"])
}

func TestE2E_CreateClient_MissingName_Returns400(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	resp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_CreateClient_NoToken_Returns401(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/users/me/clients"), map[string]string{
		"name": "Acme Corp",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── List Clients ────────────────────────────────────────────────────────────

func TestE2E_ListClients_Success(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	// Create two clients
	resp1 := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{"name": "Client A"})
	require.Equal(t, http.StatusCreated, resp1.StatusCode)
	resp1.Body.Close()

	resp2 := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{"name": "Client B"})
	require.Equal(t, http.StatusCreated, resp2.StatusCode)
	resp2.Body.Close()

	// List clients
	resp := helpers.AuthGetJSON(t, url("/users/me/clients"), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := helpers.ParseBody(t, resp)
	clients := body["data"].([]any)
	assert.Len(t, clients, 2)
}

func TestE2E_ListClients_Empty(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	resp := helpers.AuthGetJSON(t, url("/users/me/clients"), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := helpers.ParseBody(t, resp)
	clients := body["data"].([]any)
	assert.Empty(t, clients)
}

func TestE2E_ListClients_DoesNotReturnOtherUsersClients(t *testing.T) {
	clean(t)
	user1ID, token1 := createUserForClients(t, "Juan", "juan@example.com")
	_, token2 := createUserForClients(t, "Pedro", "pedro@example.com")

	// Each user creates a client with the same name
	resp1 := helpers.AuthPostJSON(t, url("/users/me/clients"), token1, map[string]string{"name": "Shared Name"})
	require.Equal(t, http.StatusCreated, resp1.StatusCode)
	resp1.Body.Close()

	resp2 := helpers.AuthPostJSON(t, url("/users/me/clients"), token2, map[string]string{"name": "Shared Name"})
	require.Equal(t, http.StatusCreated, resp2.StatusCode)
	resp2.Body.Close()

	// User 1 should only see their own client
	resp := helpers.AuthGetJSON(t, url("/users/me/clients"), token1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := helpers.ParseBody(t, resp)
	clients := body["data"].([]any)
	assert.Len(t, clients, 1)
	assert.Equal(t, user1ID, clients[0].(map[string]any)["user_id"])
}

// ─── Get Client ──────────────────────────────────────────────────────────────

func TestE2E_GetClient_Success(t *testing.T) {
	clean(t)
	userID, token := createUserForClients(t, "Juan", "juan@example.com")

	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{"name": "Acme Corp"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)

	resp := helpers.AuthGetJSON(t, url("/users/me/clients/"+clientID), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.Equal(t, clientID, body["id"])
	assert.Equal(t, "Acme Corp", body["name"])
	assert.Equal(t, userID, body["user_id"])
}

func TestE2E_GetClient_NotFound_Returns404(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	resp := helpers.AuthGetJSON(t, url("/users/me/clients/00000000-0000-0000-0000-000000000000"), token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_GetClient_InvalidID_Returns400(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	resp := helpers.AuthGetJSON(t, url("/users/me/clients/not-a-uuid"), token)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_GetClient_WrongUser_Returns403(t *testing.T) {
	clean(t)
	_, token1 := createUserForClients(t, "Juan", "juan@example.com")
	_, token2 := createUserForClients(t, "Pedro", "pedro@example.com")

	// Create client for user1
	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), token1, map[string]string{"name": "Private Client"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)

	// Try to access from user2's token
	resp := helpers.AuthGetJSON(t, url("/users/me/clients/"+clientID), token2)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// ─── Update Client ───────────────────────────────────────────────────────────

func TestE2E_UpdateClient_Success(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{"name": "Old Name"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)

	resp := helpers.AuthPutJSON(t, url("/users/me/clients/"+clientID), token, map[string]string{
		"name": "New Name",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Equal(t, "New Name", body["name"])
	assert.Equal(t, clientID, body["id"])
}

func TestE2E_UpdateClient_NotFound_Returns404(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	resp := helpers.AuthPutJSON(t, url("/users/me/clients/00000000-0000-0000-0000-000000000000"), token, map[string]string{
		"name": "Ghost",
	})

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_UpdateClient_WrongUser_Returns403(t *testing.T) {
	clean(t)
	_, token1 := createUserForClients(t, "Juan", "juan@example.com")
	_, token2 := createUserForClients(t, "Pedro", "pedro@example.com")

	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), token1, map[string]string{"name": "Private"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)

	resp := helpers.AuthPutJSON(t, url("/users/me/clients/"+clientID), token2, map[string]string{
		"name": "Stolen",
	})

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// ─── Delete Client ───────────────────────────────────────────────────────────

func TestE2E_DeleteClient_Success(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{"name": "To Delete"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)

	resp := helpers.AuthDeleteJSON(t, url("/users/me/clients/"+clientID), token)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify it's gone
	getResp := helpers.AuthGetJSON(t, url("/users/me/clients/"+clientID), token)
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestE2E_DeleteClient_NotFound_Returns404(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	resp := helpers.AuthDeleteJSON(t, url("/users/me/clients/00000000-0000-0000-0000-000000000000"), token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_DeleteClient_WrongUser_Returns403(t *testing.T) {
	clean(t)
	_, token1 := createUserForClients(t, "Juan", "juan@example.com")
	_, token2 := createUserForClients(t, "Pedro", "pedro@example.com")

	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), token1, map[string]string{"name": "Private"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)

	resp := helpers.AuthDeleteJSON(t, url("/users/me/clients/"+clientID), token2)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// ─── Full Flow ───────────────────────────────────────────────────────────────

func TestE2E_Client_FullFlow_Create_Get_Update_List_Delete(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Ana", "ana@example.com")

	// 1. Create
	createResp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{
		"name": "Acme Corp",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	clientID := created["id"].(string)
	assert.Equal(t, "Acme Corp", created["name"])

	// 2. Get — verify persisted
	getResp := helpers.AuthGetJSON(t, url("/users/me/clients/"+clientID), token)
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	fetched := helpers.ParseBody(t, getResp)
	assert.Equal(t, "Acme Corp", fetched["name"])

	// 3. Update
	updateResp := helpers.AuthPutJSON(t, url("/users/me/clients/"+clientID), token, map[string]string{
		"name": "Acme Inc",
	})
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	updated := helpers.ParseBody(t, updateResp)
	assert.Equal(t, "Acme Inc", updated["name"])

	// 4. List — should have 1 client
	listResp := helpers.AuthGetJSON(t, url("/users/me/clients"), token)
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	listBody := helpers.ParseBody(t, listResp)
	clients := listBody["data"].([]any)
	assert.Len(t, clients, 1)
	assert.Equal(t, "Acme Inc", clients[0].(map[string]any)["name"])

	// 5. Delete
	deleteResp := helpers.AuthDeleteJSON(t, url("/users/me/clients/"+clientID), token)
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode)

	// 6. List — should be empty now
	listResp2 := helpers.AuthGetJSON(t, url("/users/me/clients"), token)
	assert.Equal(t, http.StatusOK, listResp2.StatusCode)
	listBody2 := helpers.ParseBody(t, listResp2)
	clientsAfter := listBody2["data"].([]any)
	assert.Empty(t, clientsAfter)
}
