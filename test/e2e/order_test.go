package e2e

import (
	"net/http"
	"testing"

	"ductifact/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Helper: register user, create client, create project → (userID, clientID, projectID, token) ---

func createUserClientAndProject(t *testing.T, userName, email, clientName, projectName string) (string, string, string, string) {
	t.Helper()
	userID, clientID, token := createUserAndClient(t, userName, email, clientName)

	resp := helpers.AuthPostJSON(t, projURL(clientID), token, map[string]string{
		"name": projectName,
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	projectID := body["id"].(string)

	return userID, clientID, projectID, token
}

// orderURL builds the full URL for order collection endpoints under a project.
func orderURL(projectID string) string {
	return url("/projects/" + projectID + "/orders")
}

// orderItemURL builds the full URL for a single order endpoint.
func orderItemURL(orderID string) string {
	return url("/orders/" + orderID)
}

// ─── Create Order ────────────────────────────────────────────────────────────

func TestE2E_CreateOrder_Success(t *testing.T) {
	clean(t)
	_, _, projectID, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	resp := helpers.AuthPostJSON(t, orderURL(projectID), token, map[string]string{
		"title":       "Steel beams – lot 3",
		"description": "First batch of structural steel",
	})

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.NotEmpty(t, body["id"])
	assert.Equal(t, "Steel beams – lot 3", body["title"])
	assert.Equal(t, "pending", body["status"])
	assert.Equal(t, "First batch of structural steel", body["description"])
	assert.Equal(t, projectID, body["project_id"])
}

func TestE2E_CreateOrder_WithStatus_Success(t *testing.T) {
	clean(t)
	_, _, projectID, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	resp := helpers.AuthPostJSON(t, orderURL(projectID), token, map[string]string{
		"title":  "Concrete mix – delivery 1",
		"status": "completed",
	})

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Equal(t, "completed", body["status"])
}

func TestE2E_CreateOrder_MissingTitle_Returns400(t *testing.T) {
	clean(t)
	_, _, projectID, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	resp := helpers.AuthPostJSON(t, orderURL(projectID), token, map[string]string{})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_CreateOrder_InvalidStatus_Returns400(t *testing.T) {
	clean(t)
	_, _, projectID, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	resp := helpers.AuthPostJSON(t, orderURL(projectID), token, map[string]string{
		"title":  "Some order",
		"status": "invalid",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_CreateOrder_NoToken_Returns401(t *testing.T) {
	clean(t)
	_, _, projectID, _ := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	resp := helpers.PostJSON(t, orderURL(projectID), map[string]string{
		"title": "Order",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_CreateOrder_NonExistingProject_Returns404(t *testing.T) {
	clean(t)
	_, _, _, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	resp := helpers.AuthPostJSON(t, orderURL("00000000-0000-0000-0000-000000000000"), token, map[string]string{
		"title": "Order",
	})

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_CreateOrder_WrongUser_Returns404(t *testing.T) {
	clean(t)
	_, _, projectID, _ := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")
	_, _, _, token2 := createUserClientAndProject(t, "Pedro", "pedro@example.com", "Other Corp", "Other Proj")

	resp := helpers.AuthPostJSON(t, orderURL(projectID), token2, map[string]string{
		"title": "Order",
	})

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ─── List Orders ─────────────────────────────────────────────────────────────

func TestE2E_ListOrders_Success(t *testing.T) {
	clean(t)
	_, _, projectID, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	r1 := helpers.AuthPostJSON(t, orderURL(projectID), token, map[string]string{"title": "Order A"})
	require.Equal(t, http.StatusCreated, r1.StatusCode)
	r1.Body.Close()

	r2 := helpers.AuthPostJSON(t, orderURL(projectID), token, map[string]string{"title": "Order B"})
	require.Equal(t, http.StatusCreated, r2.StatusCode)
	r2.Body.Close()

	resp := helpers.AuthGetJSON(t, orderURL(projectID), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := helpers.ParseBody(t, resp)
	orders := body["data"].([]any)
	assert.Len(t, orders, 2)
}

func TestE2E_ListOrders_Empty(t *testing.T) {
	clean(t)
	_, _, projectID, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	resp := helpers.AuthGetJSON(t, orderURL(projectID), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := helpers.ParseBody(t, resp)
	orders := body["data"].([]any)
	assert.Empty(t, orders)
}

func TestE2E_ListOrders_DoesNotReturnOtherProjectsOrders(t *testing.T) {
	clean(t)
	_, clientID, projectID1, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	// Create a second project for the same client
	resp := helpers.AuthPostJSON(t, projURL(clientID), token, map[string]string{"name": "Solar Park"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	proj2Body := helpers.ParseBody(t, resp)
	projectID2 := proj2Body["id"].(string)

	// Create an order under each project
	r1 := helpers.AuthPostJSON(t, orderURL(projectID1), token, map[string]string{"title": "Order A"})
	require.Equal(t, http.StatusCreated, r1.StatusCode)
	r1.Body.Close()

	r2 := helpers.AuthPostJSON(t, orderURL(projectID2), token, map[string]string{"title": "Order B"})
	require.Equal(t, http.StatusCreated, r2.StatusCode)
	r2.Body.Close()

	// Listing orders under project1 should only return Order A
	listResp := helpers.AuthGetJSON(t, orderURL(projectID1), token)
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	listBody := helpers.ParseBody(t, listResp)
	orders := listBody["data"].([]any)
	assert.Len(t, orders, 1)
	assert.Equal(t, "Order A", orders[0].(map[string]any)["title"])
}

// ─── Get Order ───────────────────────────────────────────────────────────────

func TestE2E_GetOrder_Success(t *testing.T) {
	clean(t)
	_, _, projectID, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	createResp := helpers.AuthPostJSON(t, orderURL(projectID), token, map[string]string{
		"title":       "Steel beams – lot 3",
		"description": "First batch",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	orderID := created["id"].(string)

	resp := helpers.AuthGetJSON(t, orderItemURL(orderID), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.Equal(t, orderID, body["id"])
	assert.Equal(t, "Steel beams – lot 3", body["title"])
	assert.Equal(t, "pending", body["status"])
	assert.Equal(t, "First batch", body["description"])
	assert.Equal(t, projectID, body["project_id"])
}

func TestE2E_GetOrder_NotFound_Returns404(t *testing.T) {
	clean(t)
	_, _, _, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	resp := helpers.AuthGetJSON(t, orderItemURL("00000000-0000-0000-0000-000000000000"), token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_GetOrder_InvalidID_Returns400(t *testing.T) {
	clean(t)
	_, _, _, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	resp := helpers.AuthGetJSON(t, orderItemURL("not-a-uuid"), token)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_GetOrder_WrongUser_Returns404(t *testing.T) {
	clean(t)
	_, _, projectID1, token1 := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")
	_, _, _, token2 := createUserClientAndProject(t, "Pedro", "pedro@example.com", "Other Corp", "Other Proj")

	createResp := helpers.AuthPostJSON(t, orderURL(projectID1), token1, map[string]string{"title": "Private Order"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	orderID := created["id"].(string)

	resp := helpers.AuthGetJSON(t, orderItemURL(orderID), token2)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ─── Update Order ────────────────────────────────────────────────────────────

func TestE2E_UpdateOrder_Success(t *testing.T) {
	clean(t)
	_, _, projectID, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	createResp := helpers.AuthPostJSON(t, orderURL(projectID), token, map[string]string{
		"title": "Old Title",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	orderID := created["id"].(string)

	resp := helpers.AuthPutJSON(t, orderItemURL(orderID), token, map[string]string{
		"title":       "New Title",
		"status":      "completed",
		"description": "Updated description",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Equal(t, "New Title", body["title"])
	assert.Equal(t, "completed", body["status"])
	assert.Equal(t, "Updated description", body["description"])
	assert.Equal(t, orderID, body["id"])
}

func TestE2E_UpdateOrder_PartialUpdate_Success(t *testing.T) {
	clean(t)
	_, _, projectID, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	createResp := helpers.AuthPostJSON(t, orderURL(projectID), token, map[string]string{
		"title": "Original Title",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	orderID := created["id"].(string)

	// Only update status — title should remain unchanged
	resp := helpers.AuthPutJSON(t, orderItemURL(orderID), token, map[string]string{
		"status": "completed",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Equal(t, "Original Title", body["title"])
	assert.Equal(t, "completed", body["status"])
}

func TestE2E_UpdateOrder_NotFound_Returns404(t *testing.T) {
	clean(t)
	_, _, _, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	resp := helpers.AuthPutJSON(t, orderItemURL("00000000-0000-0000-0000-000000000000"), token, map[string]string{
		"title": "Ghost",
	})

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_UpdateOrder_WrongUser_Returns404(t *testing.T) {
	clean(t)
	_, _, projectID1, token1 := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")
	_, _, _, token2 := createUserClientAndProject(t, "Pedro", "pedro@example.com", "Other Corp", "Other Proj")

	createResp := helpers.AuthPostJSON(t, orderURL(projectID1), token1, map[string]string{"title": "Private"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	orderID := created["id"].(string)

	resp := helpers.AuthPutJSON(t, orderItemURL(orderID), token2, map[string]string{
		"title": "Stolen",
	})

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_UpdateOrder_InvalidStatus_Returns400(t *testing.T) {
	clean(t)
	_, _, projectID, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	createResp := helpers.AuthPostJSON(t, orderURL(projectID), token, map[string]string{"title": "Order"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	orderID := created["id"].(string)

	resp := helpers.AuthPutJSON(t, orderItemURL(orderID), token, map[string]string{
		"status": "invalid",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Delete Order ────────────────────────────────────────────────────────────

func TestE2E_DeleteOrder_Success(t *testing.T) {
	clean(t)
	_, _, projectID, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	createResp := helpers.AuthPostJSON(t, orderURL(projectID), token, map[string]string{"title": "To Delete"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	orderID := created["id"].(string)

	resp := helpers.AuthDeleteJSON(t, orderItemURL(orderID), token)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify it's gone
	getResp := helpers.AuthGetJSON(t, orderItemURL(orderID), token)
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestE2E_DeleteOrder_NotFound_Returns404(t *testing.T) {
	clean(t)
	_, _, _, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	resp := helpers.AuthDeleteJSON(t, orderItemURL("00000000-0000-0000-0000-000000000000"), token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_DeleteOrder_WrongUser_Returns404(t *testing.T) {
	clean(t)
	_, _, projectID1, token1 := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")
	_, _, _, token2 := createUserClientAndProject(t, "Pedro", "pedro@example.com", "Other Corp", "Other Proj")

	createResp := helpers.AuthPostJSON(t, orderURL(projectID1), token1, map[string]string{"title": "Private"})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	orderID := created["id"].(string)

	resp := helpers.AuthDeleteJSON(t, orderItemURL(orderID), token2)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ─── Full Flow ───────────────────────────────────────────────────────────────

func TestE2E_Order_FullFlow_Create_Get_Update_List_Delete(t *testing.T) {
	clean(t)
	_, _, projectID, token := createUserClientAndProject(t, "Ana", "ana@example.com", "Acme Corp", "Tower B")

	// 1. Create with default status
	createResp := helpers.AuthPostJSON(t, orderURL(projectID), token, map[string]string{
		"title":       "Steel beams – lot 3",
		"description": "First batch of structural steel",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	orderID := created["id"].(string)
	assert.Equal(t, "Steel beams – lot 3", created["title"])
	assert.Equal(t, "pending", created["status"])
	assert.Equal(t, "First batch of structural steel", created["description"])
	assert.Equal(t, projectID, created["project_id"])

	// 2. Get — verify persisted
	getResp := helpers.AuthGetJSON(t, orderItemURL(orderID), token)
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	fetched := helpers.ParseBody(t, getResp)
	assert.Equal(t, "Steel beams – lot 3", fetched["title"])
	assert.Equal(t, "pending", fetched["status"])
	assert.Equal(t, "First batch of structural steel", fetched["description"])

	// 3. Update title + status + description
	updateResp := helpers.AuthPutJSON(t, orderItemURL(orderID), token, map[string]string{
		"title":       "Steel beams – lot 3 (updated)",
		"status":      "completed",
		"description": "Updated description",
	})
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	updated := helpers.ParseBody(t, updateResp)
	assert.Equal(t, "Steel beams – lot 3 (updated)", updated["title"])
	assert.Equal(t, "completed", updated["status"])
	assert.Equal(t, "Updated description", updated["description"])

	// 4. List — should have 1 order
	listResp := helpers.AuthGetJSON(t, orderURL(projectID), token)
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	listBody := helpers.ParseBody(t, listResp)
	orders := listBody["data"].([]any)
	assert.Len(t, orders, 1)
	assert.Equal(t, "Steel beams – lot 3 (updated)", orders[0].(map[string]any)["title"])

	// 5. Delete
	deleteResp := helpers.AuthDeleteJSON(t, orderItemURL(orderID), token)
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode)

	// 6. List — should be empty now
	listResp2 := helpers.AuthGetJSON(t, orderURL(projectID), token)
	assert.Equal(t, http.StatusOK, listResp2.StatusCode)
	listBody2 := helpers.ParseBody(t, listResp2)
	ordersAfter := listBody2["data"].([]any)
	assert.Empty(t, ordersAfter)
}
