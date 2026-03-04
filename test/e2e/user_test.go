package e2e

import (
	"net/http"
	"testing"

	"ductifact/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Health ──────────────────────────────────────────────────────────────────

func TestE2E_Health(t *testing.T) {
	clean(t)

	resp := helpers.GetJSON(t, url("/health"))

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Equal(t, "healthy !!!!", body["status"])
}

// ─── Create User ─────────────────────────────────────────────────────────────

func TestE2E_CreateUser_Success(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/users"), map[string]string{
		"name":  "Juan",
		"email": "juan@example.com",
	})

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.NotEmpty(t, body["id"])
	assert.Equal(t, "Juan", body["name"])
	assert.Equal(t, "juan@example.com", body["email"])
}

func TestE2E_CreateUser_MissingName_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/users"), map[string]string{
		"email": "juan@example.com",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_CreateUser_InvalidEmail_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/users"), map[string]string{
		"name":  "Juan",
		"email": "not-an-email",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestE2E_CreateUser_DuplicateEmail_Returns409(t *testing.T) {
	clean(t)

	// First user
	resp := helpers.PostJSON(t, url("/users"), map[string]string{
		"name":  "Juan",
		"email": "same@example.com",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Second user with same email
	resp = helpers.PostJSON(t, url("/users"), map[string]string{
		"name":  "Pedro",
		"email": "same@example.com",
	})

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "email already in use")
}

func TestE2E_CreateUser_EmptyBody_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PostJSON(t, url("/users"), map[string]string{})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Get User ────────────────────────────────────────────────────────────────

func TestE2E_GetUser_Success(t *testing.T) {
	clean(t)

	// Create a user first
	createResp := helpers.PostJSON(t, url("/users"), map[string]string{
		"name":  "María",
		"email": "maria@example.com",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	id := created["id"].(string)

	// Get the user
	resp := helpers.GetJSON(t, url("/users/"+id))

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.Equal(t, id, body["id"])
	assert.Equal(t, "María", body["name"])
	assert.Equal(t, "maria@example.com", body["email"])
}

func TestE2E_GetUser_NotFound_Returns404(t *testing.T) {
	clean(t)

	resp := helpers.GetJSON(t, url("/users/00000000-0000-0000-0000-000000000000"))

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "user not found")
}

func TestE2E_GetUser_InvalidID_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.GetJSON(t, url("/users/not-a-uuid"))

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "invalid user ID")
}

// ─── Update User ─────────────────────────────────────────────────────────────

func TestE2E_UpdateUser_Name_Success(t *testing.T) {
	clean(t)

	// Create user
	createResp := helpers.PostJSON(t, url("/users"), map[string]string{
		"name":  "Juan",
		"email": "juan@example.com",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	id := created["id"].(string)

	// Update name only
	resp := helpers.PutJSON(t, url("/users/"+id), map[string]string{
		"name": "Juan Carlos",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.Equal(t, id, body["id"])
	assert.Equal(t, "Juan Carlos", body["name"])
	assert.Equal(t, "juan@example.com", body["email"]) // email unchanged
}

func TestE2E_UpdateUser_Email_Success(t *testing.T) {
	clean(t)

	// Create user
	createResp := helpers.PostJSON(t, url("/users"), map[string]string{
		"name":  "Juan",
		"email": "old@example.com",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	id := created["id"].(string)

	// Update email only
	resp := helpers.PutJSON(t, url("/users/"+id), map[string]string{
		"email": "new@example.com",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.Equal(t, "Juan", body["name"]) // name unchanged
	assert.Equal(t, "new@example.com", body["email"])
}

func TestE2E_UpdateUser_DuplicateEmail_Returns409(t *testing.T) {
	clean(t)

	// Create two users
	resp1 := helpers.PostJSON(t, url("/users"), map[string]string{
		"name": "Juan", "email": "juan@example.com",
	})
	require.Equal(t, http.StatusCreated, resp1.StatusCode)
	resp1.Body.Close()

	resp2 := helpers.PostJSON(t, url("/users"), map[string]string{
		"name": "Pedro", "email": "pedro@example.com",
	})
	require.Equal(t, http.StatusCreated, resp2.StatusCode)
	created := helpers.ParseBody(t, resp2)
	pedroID := created["id"].(string)

	// Try to update Pedro's email to Juan's
	resp := helpers.PutJSON(t, url("/users/"+pedroID), map[string]string{
		"email": "juan@example.com",
	})

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "email already in use")
}

func TestE2E_UpdateUser_NotFound_Returns404(t *testing.T) {
	clean(t)

	resp := helpers.PutJSON(t, url("/users/00000000-0000-0000-0000-000000000000"), map[string]string{
		"name": "Ghost",
	})

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestE2E_UpdateUser_InvalidID_Returns400(t *testing.T) {
	clean(t)

	resp := helpers.PutJSON(t, url("/users/not-a-uuid"), map[string]string{
		"name": "Test",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Full Flow ───────────────────────────────────────────────────────────────

func TestE2E_FullFlow_Create_Get_Update_Get(t *testing.T) {
	clean(t)

	// 1. Create
	createResp := helpers.PostJSON(t, url("/users"), map[string]string{
		"name":  "Ana",
		"email": "ana@example.com",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	id := created["id"].(string)

	assert.Equal(t, "Ana", created["name"])
	assert.Equal(t, "ana@example.com", created["email"])

	// 2. Get — verify persisted
	getResp1 := helpers.GetJSON(t, url("/users/"+id))
	assert.Equal(t, http.StatusOK, getResp1.StatusCode)
	fetched1 := helpers.ParseBody(t, getResp1)

	assert.Equal(t, id, fetched1["id"])
	assert.Equal(t, "Ana", fetched1["name"])
	assert.Equal(t, "ana@example.com", fetched1["email"])

	// 3. Update name and email
	updateResp := helpers.PutJSON(t, url("/users/"+id), map[string]string{
		"name":  "Ana María",
		"email": "anamaria@example.com",
	})
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	updated := helpers.ParseBody(t, updateResp)

	assert.Equal(t, id, updated["id"])
	assert.Equal(t, "Ana María", updated["name"])
	assert.Equal(t, "anamaria@example.com", updated["email"])

	// 4. Get — verify update persisted
	getResp2 := helpers.GetJSON(t, url("/users/"+id))
	assert.Equal(t, http.StatusOK, getResp2.StatusCode)
	fetched2 := helpers.ParseBody(t, getResp2)

	assert.Equal(t, id, fetched2["id"])
	assert.Equal(t, "Ana María", fetched2["name"])
	assert.Equal(t, "anamaria@example.com", fetched2["email"])
}
