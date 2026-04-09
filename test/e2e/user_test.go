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

	resp := helpers.GetJSON(t, rootURL("/health"))

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Equal(t, "healthy", body["status"])
}

// registerUser is a helper that registers a user and returns (id, token).
func registerUser(t *testing.T, name, email, password string) (string, string) {
	t.Helper()
	resp := helpers.PostJSON(t, url("/auth/register"), map[string]string{
		"name":     name,
		"email":    email,
		"password": password,
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	user := body["user"].(map[string]any)
	return user["id"].(string), body["access_token"].(string)
}

// ─── Get User (via /users/me) ────────────────────────────────────────────────

func TestE2E_GetMe_Success(t *testing.T) {
	clean(t)

	id, token := registerUser(t, "María", "maria@example.com", "securepass123")

	resp := helpers.AuthGetJSON(t, url("/users/me"), token)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.Equal(t, id, body["id"])
	assert.Equal(t, "María", body["name"])
	assert.Equal(t, "maria@example.com", body["email"])
}

func TestE2E_GetMe_NoToken_Returns401(t *testing.T) {
	clean(t)

	resp := helpers.GetJSON(t, url("/users/me"))

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_GetMe_InvalidToken_Returns401(t *testing.T) {
	clean(t)

	resp := helpers.AuthGetJSON(t, url("/users/me"), "invalid-token")

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── Update User (via /users/me) ─────────────────────────────────────────────

func TestE2E_UpdateMe_Name_Success(t *testing.T) {
	clean(t)

	id, token := registerUser(t, "Juan", "juan@example.com", "securepass123")

	resp := helpers.AuthPutJSON(t, url("/users/me"), token, map[string]string{
		"name": "Juan Carlos",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.Equal(t, id, body["id"])
	assert.Equal(t, "Juan Carlos", body["name"])
	assert.Equal(t, "juan@example.com", body["email"]) // email unchanged
}

func TestE2E_UpdateMe_Email_Success(t *testing.T) {
	clean(t)

	_, token := registerUser(t, "Juan", "old@example.com", "securepass123")

	resp := helpers.AuthPutJSON(t, url("/users/me"), token, map[string]string{
		"email": "new@example.com",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.Equal(t, "Juan", body["name"]) // name unchanged
	assert.Equal(t, "new@example.com", body["email"])
}

func TestE2E_UpdateMe_DuplicateEmail_Returns409(t *testing.T) {
	clean(t)

	// Create two users
	registerUser(t, "Juan", "juan@example.com", "securepass123")
	_, pedroToken := registerUser(t, "Pedro", "pedro@example.com", "securepass123")

	// Try to update Pedro's email to Juan's
	resp := helpers.AuthPutJSON(t, url("/users/me"), pedroToken, map[string]string{
		"email": "juan@example.com",
	})

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	assert.Contains(t, body["error"], "email already in use")
}

func TestE2E_UpdateMe_NoToken_Returns401(t *testing.T) {
	clean(t)

	resp := helpers.PutJSON(t, url("/users/me"), map[string]string{
		"name": "Ghost",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── Full Flow ───────────────────────────────────────────────────────────────

func TestE2E_FullFlow_Register_GetMe_UpdateMe_GetMe(t *testing.T) {
	clean(t)

	// 1. Register
	id, token := registerUser(t, "Ana", "ana@example.com", "securepass123")

	// 2. GetMe — verify persisted
	getResp1 := helpers.AuthGetJSON(t, url("/users/me"), token)
	assert.Equal(t, http.StatusOK, getResp1.StatusCode)
	fetched1 := helpers.ParseBody(t, getResp1)

	assert.Equal(t, id, fetched1["id"])
	assert.Equal(t, "Ana", fetched1["name"])
	assert.Equal(t, "ana@example.com", fetched1["email"])

	// 3. UpdateMe — name and email
	updateResp := helpers.AuthPutJSON(t, url("/users/me"), token, map[string]string{
		"name":  "Ana María",
		"email": "anamaria@example.com",
	})
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	updated := helpers.ParseBody(t, updateResp)

	assert.Equal(t, id, updated["id"])
	assert.Equal(t, "Ana María", updated["name"])
	assert.Equal(t, "anamaria@example.com", updated["email"])

	// 4. GetMe — verify update persisted
	getResp2 := helpers.AuthGetJSON(t, url("/users/me"), token)
	assert.Equal(t, http.StatusOK, getResp2.StatusCode)
	fetched2 := helpers.ParseBody(t, getResp2)

	assert.Equal(t, id, fetched2["id"])
	assert.Equal(t, "Ana María", fetched2["name"])
	assert.Equal(t, "anamaria@example.com", fetched2["email"])
}
