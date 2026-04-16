package e2e

import (
	"net/http"
	"testing"

	"ductifact/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Helpers ---

// createFullChainForPieces sets up: user → client → project → order → piece definition
// Returns (clientID, projectID, orderID, defID, token).
func createFullChainForPieces(t *testing.T) (string, string, string, string, string) {
	t.Helper()
	_, clientID, projectID, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")

	// Create order
	orderResp := helpers.AuthPostJSON(t, orderURL(clientID, projectID), token, map[string]string{
		"title": "Steel beams – lot 3",
	})
	require.Equal(t, http.StatusCreated, orderResp.StatusCode)
	orderBody := helpers.ParseBody(t, orderResp)
	orderID := orderBody["id"].(string)

	// Create piece definition
	defID := createPieceDef(t, token, "Rectangle", []string{"Length", "Width"})

	return clientID, projectID, orderID, defID, token
}

// pieceURL builds the full URL for piece endpoints under an order.
func pieceURL(clientID, projectID, orderID string, extra ...string) string {
	base := "/users/me/clients/" + clientID + "/projects/" + projectID + "/orders/" + orderID + "/pieces"
	if len(extra) > 0 {
		base += "/" + extra[0]
	}
	return url(base)
}

// ─── Create Piece ────────────────────────────────────────────────────────────

func TestE2E_CreatePiece_Success(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)

	resp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title":         "Side panel",
		"definition_id": defID,
		"dimensions":    map[string]float64{"Length": 150.5, "Width": 80.0},
		"quantity":      15,
	})

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.NotEmpty(t, body["id"])
	assert.Equal(t, "Side panel", body["title"])
	assert.Equal(t, defID, body["definition_id"])
	assert.Equal(t, float64(15), body["quantity"])
	assert.Equal(t, orderID, body["order_id"])

	dims := body["dimensions"].(map[string]any)
	assert.Equal(t, 150.5, dims["Length"])
	assert.Equal(t, 80.0, dims["Width"])
}

func TestE2E_CreatePiece_MissingTitle_Returns400(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)

	resp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"definition_id": defID,
		"dimensions":    map[string]float64{"Length": 100, "Width": 50},
		"quantity":      1,
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_CreatePiece_UnexpectedDimensions_Returns400(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)

	resp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title":         "Panel",
		"definition_id": defID,
		"dimensions":    map[string]float64{"Length": 100, "Width": 50, "Height": 30},
		"quantity":      1,
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_CreatePiece_MissingDimensions_Returns400(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)

	resp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title":         "Panel",
		"definition_id": defID,
		"dimensions":    map[string]float64{"Length": 100},
		"quantity":      1,
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_CreatePiece_NoToken_Returns401(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, _ := createFullChainForPieces(t)

	resp := helpers.PostJSON(t, pieceURL(clientID, projectID, orderID), map[string]any{
		"title":         "Panel",
		"definition_id": defID,
		"dimensions":    map[string]float64{"Length": 100, "Width": 50},
		"quantity":      1,
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_CreatePiece_WrongUser_Returns404(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, _ := createFullChainForPieces(t)
	_, _, _, token2 := createUserClientAndProject(t, "Pedro", "pedro@example.com", "Other Corp", "Other Proj")

	resp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token2, map[string]any{
		"title":         "Panel",
		"definition_id": defID,
		"dimensions":    map[string]float64{"Length": 100, "Width": 50},
		"quantity":      1,
	})

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_CreatePiece_NonExistingOrder_Returns404(t *testing.T) {
	clean(t)
	_, clientID, projectID, token := createUserClientAndProject(t, "Juan", "juan@example.com", "Acme Corp", "Tower B")
	defID := createPieceDef(t, token, "Rect", []string{"Length", "Width"})

	resp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, "00000000-0000-0000-0000-000000000000"), token, map[string]any{
		"title":         "Panel",
		"definition_id": defID,
		"dimensions":    map[string]float64{"Length": 100, "Width": 50},
		"quantity":      1,
	})

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ─── List Pieces ─────────────────────────────────────────────────────────────

func TestE2E_ListPieces_Success(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)

	r1 := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title": "Panel A", "definition_id": defID,
		"dimensions": map[string]float64{"Length": 100, "Width": 50}, "quantity": 1,
	})
	require.Equal(t, http.StatusCreated, r1.StatusCode)
	r1.Body.Close()

	r2 := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title": "Panel B", "definition_id": defID,
		"dimensions": map[string]float64{"Length": 200, "Width": 100}, "quantity": 2,
	})
	require.Equal(t, http.StatusCreated, r2.StatusCode)
	r2.Body.Close()

	resp := helpers.AuthGetJSON(t, pieceURL(clientID, projectID, orderID), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := helpers.ParseBody(t, resp)
	pieces := body["data"].([]any)
	assert.Len(t, pieces, 2)
}

func TestE2E_ListPieces_Empty(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, _, token := createFullChainForPieces(t)

	resp := helpers.AuthGetJSON(t, pieceURL(clientID, projectID, orderID), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := helpers.ParseBody(t, resp)
	pieces := body["data"].([]any)
	assert.Empty(t, pieces)
}

// ─── Get Piece ───────────────────────────────────────────────────────────────

func TestE2E_GetPiece_Success(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)

	createResp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title": "Side panel", "definition_id": defID,
		"dimensions": map[string]float64{"Length": 150.5, "Width": 80.0}, "quantity": 15,
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	pieceID := created["id"].(string)

	resp := helpers.AuthGetJSON(t, pieceURL(clientID, projectID, orderID, pieceID), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.Equal(t, pieceID, body["id"])
	assert.Equal(t, "Side panel", body["title"])
	assert.Equal(t, orderID, body["order_id"])
}

func TestE2E_GetPiece_NotFound_Returns404(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, _, token := createFullChainForPieces(t)

	resp := helpers.AuthGetJSON(t, pieceURL(clientID, projectID, orderID, "00000000-0000-0000-0000-000000000000"), token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_GetPiece_InvalidID_Returns400(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, _, token := createFullChainForPieces(t)

	resp := helpers.AuthGetJSON(t, pieceURL(clientID, projectID, orderID, "not-a-uuid"), token)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_GetPiece_WrongUser_Returns404(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)
	_, _, _, token2 := createUserClientAndProject(t, "Pedro", "pedro@example.com", "Other Corp", "Other Proj")

	createResp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title": "Private Piece", "definition_id": defID,
		"dimensions": map[string]float64{"Length": 100, "Width": 50}, "quantity": 1,
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	pieceID := created["id"].(string)

	resp := helpers.AuthGetJSON(t, pieceURL(clientID, projectID, orderID, pieceID), token2)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ─── Update Piece ────────────────────────────────────────────────────────────

func TestE2E_UpdatePiece_Success(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)

	createResp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title": "Old Title", "definition_id": defID,
		"dimensions": map[string]float64{"Length": 100, "Width": 50}, "quantity": 1,
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	pieceID := created["id"].(string)

	resp := helpers.AuthPutJSON(t, pieceURL(clientID, projectID, orderID, pieceID), token, map[string]any{
		"title":      "New Title",
		"dimensions": map[string]float64{"Length": 200, "Width": 100},
		"quantity":   25,
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Equal(t, "New Title", body["title"])
	assert.Equal(t, float64(25), body["quantity"])
	dims := body["dimensions"].(map[string]any)
	assert.Equal(t, 200.0, dims["Length"])
	assert.Equal(t, 100.0, dims["Width"])
}

func TestE2E_UpdatePiece_PartialUpdate_Success(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)

	createResp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title": "Original Title", "definition_id": defID,
		"dimensions": map[string]float64{"Length": 100, "Width": 50}, "quantity": 10,
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	pieceID := created["id"].(string)

	// Only update quantity — title and dimensions should remain unchanged
	resp := helpers.AuthPutJSON(t, pieceURL(clientID, projectID, orderID, pieceID), token, map[string]any{
		"quantity": 20,
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Equal(t, "Original Title", body["title"])
	assert.Equal(t, float64(20), body["quantity"])
}

func TestE2E_UpdatePiece_InvalidDimensions_Returns400(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)

	createResp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title": "Panel", "definition_id": defID,
		"dimensions": map[string]float64{"Length": 100, "Width": 50}, "quantity": 1,
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	pieceID := created["id"].(string)

	resp := helpers.AuthPutJSON(t, pieceURL(clientID, projectID, orderID, pieceID), token, map[string]any{
		"dimensions": map[string]float64{"Length": 100, "Width": 50, "Height": 30},
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_UpdatePiece_NotFound_Returns404(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, _, token := createFullChainForPieces(t)

	resp := helpers.AuthPutJSON(t, pieceURL(clientID, projectID, orderID, "00000000-0000-0000-0000-000000000000"), token, map[string]any{
		"title": "Ghost",
	})

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_UpdatePiece_WrongUser_Returns404(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)
	_, _, _, token2 := createUserClientAndProject(t, "Pedro", "pedro@example.com", "Other Corp", "Other Proj")

	createResp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title": "Private", "definition_id": defID,
		"dimensions": map[string]float64{"Length": 100, "Width": 50}, "quantity": 1,
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	pieceID := created["id"].(string)

	resp := helpers.AuthPutJSON(t, pieceURL(clientID, projectID, orderID, pieceID), token2, map[string]any{
		"title": "Stolen",
	})

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ─── Delete Piece ────────────────────────────────────────────────────────────

func TestE2E_DeletePiece_Success(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)

	createResp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title": "To Delete", "definition_id": defID,
		"dimensions": map[string]float64{"Length": 100, "Width": 50}, "quantity": 1,
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	pieceID := created["id"].(string)

	resp := helpers.AuthDeleteJSON(t, pieceURL(clientID, projectID, orderID, pieceID), token)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify it's gone
	getResp := helpers.AuthGetJSON(t, pieceURL(clientID, projectID, orderID, pieceID), token)
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestE2E_DeletePiece_NotFound_Returns404(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, _, token := createFullChainForPieces(t)

	resp := helpers.AuthDeleteJSON(t, pieceURL(clientID, projectID, orderID, "00000000-0000-0000-0000-000000000000"), token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_DeletePiece_WrongUser_Returns404(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)
	_, _, _, token2 := createUserClientAndProject(t, "Pedro", "pedro@example.com", "Other Corp", "Other Proj")

	createResp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title": "Private", "definition_id": defID,
		"dimensions": map[string]float64{"Length": 100, "Width": 50}, "quantity": 1,
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	pieceID := created["id"].(string)

	resp := helpers.AuthDeleteJSON(t, pieceURL(clientID, projectID, orderID, pieceID), token2)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ─── Full Flow ───────────────────────────────────────────────────────────────

func TestE2E_Piece_FullFlow_Create_Get_Update_List_Delete(t *testing.T) {
	clean(t)
	clientID, projectID, orderID, defID, token := createFullChainForPieces(t)

	// 1. Create
	createResp := helpers.AuthPostJSON(t, pieceURL(clientID, projectID, orderID), token, map[string]any{
		"title":         "Side panel",
		"definition_id": defID,
		"dimensions":    map[string]float64{"Length": 150.5, "Width": 80.0},
		"quantity":      15,
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	pieceID := created["id"].(string)
	assert.Equal(t, "Side panel", created["title"])
	assert.Equal(t, float64(15), created["quantity"])

	// 2. Get
	getResp := helpers.AuthGetJSON(t, pieceURL(clientID, projectID, orderID, pieceID), token)
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	fetched := helpers.ParseBody(t, getResp)
	assert.Equal(t, "Side panel", fetched["title"])

	// 3. Update
	updateResp := helpers.AuthPutJSON(t, pieceURL(clientID, projectID, orderID, pieceID), token, map[string]any{
		"title":      "Side panel (revised)",
		"dimensions": map[string]float64{"Length": 200, "Width": 100},
		"quantity":   20,
	})
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	updated := helpers.ParseBody(t, updateResp)
	assert.Equal(t, "Side panel (revised)", updated["title"])
	assert.Equal(t, float64(20), updated["quantity"])

	// 4. List — should have 1 piece
	listResp := helpers.AuthGetJSON(t, pieceURL(clientID, projectID, orderID), token)
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	listBody := helpers.ParseBody(t, listResp)
	pieces := listBody["data"].([]any)
	assert.Len(t, pieces, 1)
	assert.Equal(t, "Side panel (revised)", pieces[0].(map[string]any)["title"])

	// 5. Delete
	deleteResp := helpers.AuthDeleteJSON(t, pieceURL(clientID, projectID, orderID, pieceID), token)
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode)

	// 6. List — should be empty
	listResp2 := helpers.AuthGetJSON(t, pieceURL(clientID, projectID, orderID), token)
	assert.Equal(t, http.StatusOK, listResp2.StatusCode)
	listBody2 := helpers.ParseBody(t, listResp2)
	piecesAfter := listBody2["data"].([]any)
	assert.Empty(t, piecesAfter)
}
