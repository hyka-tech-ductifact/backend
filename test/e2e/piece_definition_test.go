package e2e

import (
	"net/http"
	"testing"

	"ductifact/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pieceDefURL builds the full URL for piece definition endpoints.
func pieceDefURL(extra ...string) string {
	base := "/piece-definitions"
	if len(extra) > 0 {
		base += "/" + extra[0]
	}
	return url(base)
}

// createPieceDef is a helper that creates a piece definition (no image) and returns its ID.
func createPieceDef(t *testing.T, token string, name string, schema []string) string {
	t.Helper()
	resp := helpers.AuthPostMultipart(t, pieceDefURL(), token, map[string]any{
		"name":             name,
		"dimension_schema": schema,
	}, nil, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	return body["id"].(string)
}

// ─── Create PieceDefinition ──────────────────────────────────────────────────

func TestE2E_CreatePieceDefinition_Success(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	resp := helpers.AuthPostMultipart(t, pieceDefURL(), token, map[string]any{
		"name":             "Rectangle",
		"dimension_schema": []string{"Length", "Width"},
	}, nil, "")

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.NotEmpty(t, body["id"])
	assert.Equal(t, "Rectangle", body["name"])
	assert.Empty(t, body["image_url"])
	assert.Empty(t, body["thumbnail_url"])
	assert.False(t, body["predefined"].(bool))

	schema := body["dimension_schema"].([]any)
	assert.Len(t, schema, 2)
	assert.Equal(t, "Length", schema[0])
	assert.Equal(t, "Width", schema[1])
}

func TestE2E_CreatePieceDefinition_MissingName_Returns400(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	resp := helpers.AuthPostMultipart(t, pieceDefURL(), token, map[string]any{
		"dimension_schema": []string{"Length"},
	}, nil, "")

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_CreatePieceDefinition_EmptySchema_Returns400(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	resp := helpers.AuthPostMultipart(t, pieceDefURL(), token, map[string]any{
		"name":             "Bad Def",
		"dimension_schema": []string{},
	}, nil, "")

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_CreatePieceDefinition_NoToken_Returns401(t *testing.T) {
	clean(t)

	resp := helpers.PostMultipart(t, pieceDefURL(), map[string]any{
		"name":             "Rect",
		"dimension_schema": []string{"Length"},
	}, nil, "")

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── List PieceDefinitions ───────────────────────────────────────────────────

func TestE2E_ListPieceDefinitions_Success(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	createPieceDef(t, token, "Rect", []string{"Length", "Width"})
	createPieceDef(t, token, "Circle", []string{"Radius"})

	resp := helpers.AuthGetJSON(t, pieceDefURL(), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := helpers.ParseBody(t, resp)
	data := body["data"].([]any)
	assert.GreaterOrEqual(t, len(data), 2)
}

func TestE2E_ListPieceDefinitions_DoesNotReturnOtherUsersCustomDefs(t *testing.T) {
	clean(t)
	_, token1 := createUserForClients(t, "Juan", "juan@example.com")
	_, token2 := createUserForClients(t, "Pedro", "pedro@example.com")

	createPieceDef(t, token1, "Only Juan's", []string{"Length"})
	createPieceDef(t, token2, "Only Pedro's", []string{"Width"})

	// Pedro's list should not include Juan's custom def
	resp := helpers.AuthGetJSON(t, pieceDefURL(), token2)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := helpers.ParseBody(t, resp)
	data := body["data"].([]any)
	for _, d := range data {
		def := d.(map[string]any)
		if !def["predefined"].(bool) {
			assert.Equal(t, "Only Pedro's", def["name"])
		}
	}
}

// ─── Get PieceDefinition ─────────────────────────────────────────────────────

func TestE2E_GetPieceDefinition_Success(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	defID := createPieceDef(t, token, "Rectangle", []string{"Length", "Width"})

	resp := helpers.AuthGetJSON(t, pieceDefURL(defID), token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.Equal(t, defID, body["id"])
	assert.Equal(t, "Rectangle", body["name"])
}

func TestE2E_GetPieceDefinition_NotFound_Returns404(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	resp := helpers.AuthGetJSON(t, pieceDefURL("00000000-0000-0000-0000-000000000000"), token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_GetPieceDefinition_OtherUsersCustomDef_Returns404(t *testing.T) {
	clean(t)
	_, token1 := createUserForClients(t, "Juan", "juan@example.com")
	_, token2 := createUserForClients(t, "Pedro", "pedro@example.com")

	defID := createPieceDef(t, token1, "Juan's Def", []string{"Length"})

	resp := helpers.AuthGetJSON(t, pieceDefURL(defID), token2)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ─── Update PieceDefinition ──────────────────────────────────────────────────

func TestE2E_UpdatePieceDefinition_Success(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	defID := createPieceDef(t, token, "Old Name", []string{"Length"})

	resp := helpers.AuthPutMultipart(t, pieceDefURL(defID), token, map[string]any{
		"name":             "New Name",
		"dimension_schema": []string{"Height", "Radius"},
	}, nil, "")

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Equal(t, "New Name", body["name"])

	schema := body["dimension_schema"].([]any)
	assert.Len(t, schema, 2)
}

func TestE2E_UpdatePieceDefinition_NotOwned_Returns404(t *testing.T) {
	clean(t)
	_, token1 := createUserForClients(t, "Juan", "juan@example.com")
	_, token2 := createUserForClients(t, "Pedro", "pedro@example.com")

	defID := createPieceDef(t, token1, "Juan's Def", []string{"Length"})

	resp := helpers.AuthPutMultipart(t, pieceDefURL(defID), token2, map[string]any{
		"name": "Stolen",
	}, nil, "")

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ─── Delete PieceDefinition ──────────────────────────────────────────────────

func TestE2E_DeletePieceDefinition_Success(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	defID := createPieceDef(t, token, "To Delete", []string{"Length"})

	resp := helpers.AuthDeleteJSON(t, pieceDefURL(defID), token)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	getResp := helpers.AuthGetJSON(t, pieceDefURL(defID), token)
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestE2E_DeletePieceDefinition_NotOwned_Returns404(t *testing.T) {
	clean(t)
	_, token1 := createUserForClients(t, "Juan", "juan@example.com")
	_, token2 := createUserForClients(t, "Pedro", "pedro@example.com")

	defID := createPieceDef(t, token1, "Juan's Def", []string{"Length"})

	resp := helpers.AuthDeleteJSON(t, pieceDefURL(defID), token2)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ─── Full Flow ───────────────────────────────────────────────────────────────

func TestE2E_PieceDefinition_FullFlow(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Ana", "ana@example.com")

	// 1. Create
	createResp := helpers.AuthPostMultipart(t, pieceDefURL(), token, map[string]any{
		"name":             "Rectangle",
		"dimension_schema": []string{"Length", "Width"},
	}, nil, "")
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	defID := created["id"].(string)
	assert.Equal(t, "Rectangle", created["name"])

	// 2. Get
	getResp := helpers.AuthGetJSON(t, pieceDefURL(defID), token)
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	fetched := helpers.ParseBody(t, getResp)
	assert.Equal(t, "Rectangle", fetched["name"])

	// 3. Update
	updateResp := helpers.AuthPutMultipart(t, pieceDefURL(defID), token, map[string]any{
		"name":             "Updated Rectangle",
		"dimension_schema": []string{"Height", "Radius", "Depth"},
	}, nil, "")
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	updated := helpers.ParseBody(t, updateResp)
	assert.Equal(t, "Updated Rectangle", updated["name"])

	// 4. List
	listResp := helpers.AuthGetJSON(t, pieceDefURL(), token)
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	listBody := helpers.ParseBody(t, listResp)
	data := listBody["data"].([]any)
	assert.GreaterOrEqual(t, len(data), 1)

	// 5. Delete
	deleteResp := helpers.AuthDeleteJSON(t, pieceDefURL(defID), token)
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode)

	// 6. Verify gone
	getResp2 := helpers.AuthGetJSON(t, pieceDefURL(defID), token)
	assert.Equal(t, http.StatusNotFound, getResp2.StatusCode)
}
