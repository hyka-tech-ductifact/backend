package e2e

import (
	"net/http"
	"testing"

	"ductifact/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Helper: register a user, create a client, return (userID, clientID, token) ---

func createUserAndClient(t *testing.T, userName, email, clientName string) (string, string, string) {
	t.Helper()
	userID, token := createUserForClients(t, userName, email)

	resp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{
		"name": clientName,
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	clientID := body["id"].(string)

	return userID, clientID, token
}

// projURL builds the full URL for project endpoints under a client.
func projURL(clientID string, extra ...string) string {
	base := "/users/me/clients/" + clientID + "/projects"
	if len(extra) > 0 {
		base += "/" + extra[0]
	}
	return url(base)
}

// ─── Create Project ──────────────────────────────────────────────────────────

func TestE2E_CreateProject_Success(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	resp := helpers.AuthPostJSON(t, projURL(clientID), token, map[string]string{
		"name":         "Residential Tower B",
		"address":      "Calle Mayor 12, Madrid",
		"manager_name": "Carlos Pérez",
		"phone":        "+34 699 111 222",
		"description":  "14-storey residential building, phase 1",
	})

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.NotEmpty(t, body["id"])
	assert.Equal(t, "Residential Tower B", body["name"])
	assert.Equal(t, "Calle Mayor 12, Madrid", body["address"])
	assert.Equal(t, "Carlos Pérez", body["manager_name"])
	assert.Equal(t, "+34 699 111 222", body["phone"])
	assert.Equal(t, "14-storey residential building, phase 1", body["description"])
	assert.Equal(t, clientID, body["client_id"])
}

func TestE2E_CreateProject_MinimalFields_Success(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	resp := helpers.AuthPostJSON(t, projURL(clientID), token, map[string]string{
		"name": "Minimal Project",
	})

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.NotEmpty(t, body["id"])
	assert.Equal(t, "Minimal Project", body["name"])
	assert.Equal(t, "", body["address"])
	assert.Equal(t, "", body["manager_name"])
	assert.Equal(t, "", body["phone"])
	assert.Equal(t, "", body["description"])
	assert.Equal(t, clientID, body["client_id"])
}

func TestE2E_CreateProject_MissingName_Returns400(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	resp := helpers.AuthPostJSON(t, projURL(clientID), token, map[string]string{})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_CreateProject_NoToken_Returns401(t *testing.T) {
	clean(t)
	_, clientID, _ := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	resp := helpers.PostJSON(t, projURL(clientID), map[string]string{
		"name": "Tower",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_CreateProject_NonExistingClient_Returns404(t *testing.T) {
	clean(t)
	_, _, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	resp := helpers.AuthPostJSON(t, projURL("00000000-0000-0000-0000-000000000000"), token, map[string]string{
		"name": "Tower",
	})

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_CreateProject_WrongUser_Returns403(t *testing.T) {
	clean(t)
	_, clientID, _ := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")
	_, _, token2 := createUserAndClient(t, "Pedro", "pedro@example.com", "Other Corp")

	resp := helpers.AuthPostJSON(t, projURL(clientID), token2, map[string]string{
		"name": "Tower",
	})

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// ─── List Projects ───────────────────────────────────────────────────────────

func TestE2E_ListProjects_Success(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	resp1 := helpers.AuthPostJSON(t, projURL(clientID), token, map[string]string{"name": "Project A"})
	require.Equal(t, http.StatusCreated, resp1.StatusCode)
	resp1.Body.Close()

	resp2 := helpers.AuthPostJSON(t, projURL(clientID), token, map[string]string{"name": "Project B"})
	require.Equal(t, http.StatusCreated, resp2.StatusCode)
	resp2.Body.Close()

	resp := helpers.AuthGetJSON(t, projURL(clientID), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := helpers.ParseBody(t, resp)
	projects := body["data"].([]any)
	assert.Len(t, projects, 2)
}

func TestE2E_ListProjects_Empty(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	resp := helpers.AuthGetJSON(t, projURL(clientID), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := helpers.ParseBody(t, resp)
	projects := body["data"].([]any)
	assert.Empty(t, projects)
}

func TestE2E_ListProjects_DoesNotReturnOtherClientsProjects(t *testing.T) {
	clean(t)
	_, clientID1, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	// Create a second client for the same user
	resp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{"name": "Beta Inc"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	client2Body := helpers.ParseBody(t, resp)
	clientID2 := client2Body["id"].(string)

	// Create a project under each client
	r1 := helpers.AuthPostJSON(t, projURL(clientID1), token, map[string]string{"name": "Project A"})
	require.Equal(t, http.StatusCreated, r1.StatusCode)
	r1.Body.Close()

	r2 := helpers.AuthPostJSON(t, projURL(clientID2), token, map[string]string{"name": "Project B"})
	require.Equal(t, http.StatusCreated, r2.StatusCode)
	r2.Body.Close()

	// Listing projects under client1 should only return Project A
	listResp := helpers.AuthGetJSON(t, projURL(clientID1), token)
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	listBody := helpers.ParseBody(t, listResp)
	projects := listBody["data"].([]any)
	assert.Len(t, projects, 1)
	assert.Equal(t, "Project A", projects[0].(map[string]any)["name"])
}

// ─── Get Project ─────────────────────────────────────────────────────────────

func TestE2E_GetProject_Success(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	createResp := helpers.AuthPostJSON(t, projURL(clientID), token, map[string]string{
		"name":         "Residential Tower B",
		"address":      "Calle Mayor 12, Madrid",
		"manager_name": "Carlos Pérez",
		"phone":        "+34 699 111 222",
		"description":  "14-storey residential building",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	projectID := created["id"].(string)

	resp := helpers.AuthGetJSON(t, projURL(clientID, projectID), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.Equal(t, projectID, body["id"])
	assert.Equal(t, "Residential Tower B", body["name"])
	assert.Equal(t, "Calle Mayor 12, Madrid", body["address"])
	assert.Equal(t, "Carlos Pérez", body["manager_name"])
	assert.Equal(t, "+34 699 111 222", body["phone"])
	assert.Equal(t, "14-storey residential building", body["description"])
	assert.Equal(t, clientID, body["client_id"])
}

func TestE2E_GetProject_NotFound_Returns404(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	resp := helpers.AuthGetJSON(t, projURL(clientID, "00000000-0000-0000-0000-000000000000"), token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_GetProject_InvalidID_Returns400(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	resp := helpers.AuthGetJSON(t, projURL(clientID, "not-a-uuid"), token)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_GetProject_WrongUser_Returns403(t *testing.T) {
	clean(t)
	_, clientID1, token1 := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")
	_, _, token2 := createUserAndClient(t, "Pedro", "pedro@example.com", "Other Corp")

	createResp := helpers.AuthPostJSON(t, projURL(clientID1), token1, map[string]string{"name": "Private Project"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	projectID := created["id"].(string)

	// Try to access from user2's token using user1's client
	resp := helpers.AuthGetJSON(t, projURL(clientID1, projectID), token2)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// ─── Update Project ──────────────────────────────────────────────────────────

func TestE2E_UpdateProject_Success(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	createResp := helpers.AuthPostJSON(t, projURL(clientID), token, map[string]string{
		"name":    "Old Name",
		"address": "Old Address",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	projectID := created["id"].(string)

	resp := helpers.AuthPutJSON(t, projURL(clientID, projectID), token, map[string]string{
		"name":         "New Name",
		"address":      "New Address",
		"manager_name": "New Manager",
		"phone":        "+34 600 999 888",
		"description":  "Updated description",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Equal(t, "New Name", body["name"])
	assert.Equal(t, "New Address", body["address"])
	assert.Equal(t, "New Manager", body["manager_name"])
	assert.Equal(t, "+34 600 999 888", body["phone"])
	assert.Equal(t, "Updated description", body["description"])
	assert.Equal(t, projectID, body["id"])
}

func TestE2E_UpdateProject_PartialUpdate_Success(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	createResp := helpers.AuthPostJSON(t, projURL(clientID), token, map[string]string{
		"name":    "Original Name",
		"address": "Original Address",
		"phone":   "+34 600 111 222",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	projectID := created["id"].(string)

	// Only update address — other fields should remain unchanged
	resp := helpers.AuthPutJSON(t, projURL(clientID, projectID), token, map[string]string{
		"address": "Updated Address Only",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Equal(t, "Original Name", body["name"])
	assert.Equal(t, "Updated Address Only", body["address"])
	assert.Equal(t, "+34 600 111 222", body["phone"])
}

func TestE2E_UpdateProject_NotFound_Returns404(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	resp := helpers.AuthPutJSON(t, projURL(clientID, "00000000-0000-0000-0000-000000000000"), token, map[string]string{
		"name": "Ghost",
	})

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_UpdateProject_WrongUser_Returns403(t *testing.T) {
	clean(t)
	_, clientID1, token1 := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")
	_, _, token2 := createUserAndClient(t, "Pedro", "pedro@example.com", "Other Corp")

	createResp := helpers.AuthPostJSON(t, projURL(clientID1), token1, map[string]string{"name": "Private"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	projectID := created["id"].(string)

	resp := helpers.AuthPutJSON(t, projURL(clientID1, projectID), token2, map[string]string{
		"name": "Stolen",
	})

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// ─── Delete Project ──────────────────────────────────────────────────────────

func TestE2E_DeleteProject_Success(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	createResp := helpers.AuthPostJSON(t, projURL(clientID), token, map[string]string{"name": "To Delete"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	projectID := created["id"].(string)

	resp := helpers.AuthDeleteJSON(t, projURL(clientID, projectID), token)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify it's gone
	getResp := helpers.AuthGetJSON(t, projURL(clientID, projectID), token)
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestE2E_DeleteProject_NotFound_Returns404(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")

	resp := helpers.AuthDeleteJSON(t, projURL(clientID, "00000000-0000-0000-0000-000000000000"), token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_DeleteProject_WrongUser_Returns403(t *testing.T) {
	clean(t)
	_, clientID1, token1 := createUserAndClient(t, "Juan", "juan@example.com", "Acme Corp")
	_, _, token2 := createUserAndClient(t, "Pedro", "pedro@example.com", "Other Corp")

	createResp := helpers.AuthPostJSON(t, projURL(clientID1), token1, map[string]string{"name": "Private"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	projectID := created["id"].(string)

	resp := helpers.AuthDeleteJSON(t, projURL(clientID1, projectID), token2)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// ─── Full Flow ───────────────────────────────────────────────────────────────

func TestE2E_Project_FullFlow_Create_Get_Update_List_Delete(t *testing.T) {
	clean(t)
	_, clientID, token := createUserAndClient(t, "Ana", "ana@example.com", "Acme Corp")

	// 1. Create with all fields
	createResp := helpers.AuthPostJSON(t, projURL(clientID), token, map[string]string{
		"name":         "Residential Tower B",
		"address":      "Calle Mayor 12, Madrid",
		"manager_name": "Carlos Pérez",
		"phone":        "+34 699 111 222",
		"description":  "14-storey residential building, phase 1",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	projectID := created["id"].(string)
	assert.Equal(t, "Residential Tower B", created["name"])
	assert.Equal(t, "Calle Mayor 12, Madrid", created["address"])
	assert.Equal(t, "Carlos Pérez", created["manager_name"])
	assert.Equal(t, "+34 699 111 222", created["phone"])
	assert.Equal(t, "14-storey residential building, phase 1", created["description"])

	// 2. Get — verify persisted
	getResp := helpers.AuthGetJSON(t, projURL(clientID, projectID), token)
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	fetched := helpers.ParseBody(t, getResp)
	assert.Equal(t, "Residential Tower B", fetched["name"])
	assert.Equal(t, "Calle Mayor 12, Madrid", fetched["address"])
	assert.Equal(t, "Carlos Pérez", fetched["manager_name"])

	// 3. Update all fields
	updateResp := helpers.AuthPutJSON(t, projURL(clientID, projectID), token, map[string]string{
		"name":         "Tower B - Phase 2",
		"address":      "Avenida de la Constitución 1",
		"manager_name": "Ana García",
		"phone":        "+34 600 999 888",
		"description":  "Updated to phase 2",
	})
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	updated := helpers.ParseBody(t, updateResp)
	assert.Equal(t, "Tower B - Phase 2", updated["name"])
	assert.Equal(t, "Avenida de la Constitución 1", updated["address"])
	assert.Equal(t, "Ana García", updated["manager_name"])
	assert.Equal(t, "+34 600 999 888", updated["phone"])
	assert.Equal(t, "Updated to phase 2", updated["description"])

	// 4. List — should have 1 project
	listResp := helpers.AuthGetJSON(t, projURL(clientID), token)
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	listBody := helpers.ParseBody(t, listResp)
	projects := listBody["data"].([]any)
	assert.Len(t, projects, 1)
	assert.Equal(t, "Tower B - Phase 2", projects[0].(map[string]any)["name"])

	// 5. Delete
	deleteResp := helpers.AuthDeleteJSON(t, projURL(clientID, projectID), token)
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode)

	// 6. List — should be empty now
	listResp2 := helpers.AuthGetJSON(t, projURL(clientID), token)
	assert.Equal(t, http.StatusOK, listResp2.StatusCode)
	listBody2 := helpers.ParseBody(t, listResp2)
	projectsAfter := listBody2["data"].([]any)
	assert.Empty(t, projectsAfter)
}
