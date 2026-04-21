package e2e

import (
	"io"
	"net/http"
	"testing"

	"ductifact/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Create PieceDefinition with Image ───────────────────────────────────────

func TestE2E_CreatePieceDefinition_WithImage_ReturnsURLs(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	resp := helpers.AuthPostMultipart(t, pieceDefURL(), token, map[string]any{
		"name":             "With Image",
		"dimension_schema": []string{"Length"},
	}, helpers.MinimalPNG(), "photo.png")

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	imageURL, _ := body["image_url"].(string)
	thumbnailURL, _ := body["thumbnail_url"].(string)

	assert.NotEmpty(t, imageURL, "image_url should be set")
	assert.NotEmpty(t, thumbnailURL, "thumbnail_url should be set")
	assert.Contains(t, imageURL, "/v1/files/piece-definitions/")
	assert.Contains(t, imageURL, "/original.png")
	assert.Contains(t, thumbnailURL, "/v1/files/piece-definitions/")
	assert.Contains(t, thumbnailURL, "/thumb.png")
}

func TestE2E_CreatePieceDefinition_WithoutImage_EmptyURLs(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	resp := helpers.AuthPostMultipart(t, pieceDefURL(), token, map[string]any{
		"name":             "No Image",
		"dimension_schema": []string{"Width"},
	}, nil, "")

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.Empty(t, body["image_url"])
	assert.Empty(t, body["thumbnail_url"])
}

// ─── Update PieceDefinition with Image ───────────────────────────────────────

func TestE2E_UpdatePieceDefinition_WithNewImage_ReplacesURLs(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	// Create with image
	createResp := helpers.AuthPostMultipart(t, pieceDefURL(), token, map[string]any{
		"name":             "Original",
		"dimension_schema": []string{"L"},
	}, helpers.MinimalPNG(), "first.png")
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	defID := created["id"].(string)
	oldImageURL := created["image_url"].(string)

	// Update with new image
	updateResp := helpers.AuthPutMultipart(t, pieceDefURL(defID), token, map[string]any{
		"name": "Updated",
	}, helpers.MinimalPNG(), "second.png")
	require.Equal(t, http.StatusOK, updateResp.StatusCode)
	updated := helpers.ParseBody(t, updateResp)

	newImageURL := updated["image_url"].(string)
	assert.NotEmpty(t, newImageURL)
	assert.NotEqual(t, oldImageURL, newImageURL, "image URL should change after re-upload")
	assert.Contains(t, newImageURL, "/original.png")
}

func TestE2E_UpdatePieceDefinition_WithImageOnly_KeepsExistingFields(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	defID := createPieceDef(t, token, "Keep Name", []string{"L"})

	// Send only image, no "data" part
	resp := helpers.AuthPutMultipart(t, pieceDefURL(defID), token, nil, helpers.MinimalPNG(), "photo.png")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	assert.Equal(t, "Keep Name", body["name"], "name should remain unchanged")
	assert.NotEmpty(t, body["image_url"], "image should be set")
}

// ─── File Proxy ──────────────────────────────────────────────────────────────

func TestE2E_FileProxy_ServesUploadedImage(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	// Create with image
	resp := helpers.AuthPostMultipart(t, pieceDefURL(), token, map[string]any{
		"name":             "Proxy Test",
		"dimension_schema": []string{"L"},
	}, helpers.MinimalPNG(), "test.png")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)

	imageURL := body["image_url"].(string)
	thumbnailURL := body["thumbnail_url"].(string)

	// Fetch original via file proxy (public, no auth required)
	fileResp := helpers.GetJSON(t, rootURL(imageURL))
	assert.Equal(t, http.StatusOK, fileResp.StatusCode)
	assert.Equal(t, "image/png", fileResp.Header.Get("Content-Type"))
	assert.Contains(t, fileResp.Header.Get("Cache-Control"), "immutable")
	imgData, _ := io.ReadAll(fileResp.Body)
	fileResp.Body.Close()
	assert.True(t, len(imgData) > 0, "image body should not be empty")

	// Fetch thumbnail via file proxy
	thumbResp := helpers.GetJSON(t, rootURL(thumbnailURL))
	assert.Equal(t, http.StatusOK, thumbResp.StatusCode)
	assert.Contains(t, thumbResp.Header.Get("Content-Type"), "image/")
	thumbResp.Body.Close()
}

func TestE2E_FileProxy_NonexistentFile_Returns404(t *testing.T) {
	resp := helpers.GetJSON(t, rootURL("/v1/files/piece-definitions/00000000-0000-0000-0000-000000000000/original.png"))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

// ─── Delete cleans up files ──────────────────────────────────────────────────

func TestE2E_DeletePieceDefinition_WithImage_FilesNoLongerAccessible(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Juan", "juan@example.com")

	// Create with image
	resp := helpers.AuthPostMultipart(t, pieceDefURL(), token, map[string]any{
		"name":             "To Delete",
		"dimension_schema": []string{"L"},
	}, helpers.MinimalPNG(), "delete-me.png")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := helpers.ParseBody(t, resp)
	defID := body["id"].(string)
	imageURL := body["image_url"].(string)
	thumbnailURL := body["thumbnail_url"].(string)

	// Verify files are accessible before delete
	preResp := helpers.GetJSON(t, rootURL(imageURL))
	require.Equal(t, http.StatusOK, preResp.StatusCode)
	preResp.Body.Close()

	// Delete piece definition
	delResp := helpers.AuthDeleteJSON(t, pieceDefURL(defID), token)
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)

	// Verify files are cleaned up (best-effort)
	postResp := helpers.GetJSON(t, rootURL(imageURL))
	assert.Equal(t, http.StatusNotFound, postResp.StatusCode)
	postResp.Body.Close()

	postThumbResp := helpers.GetJSON(t, rootURL(thumbnailURL))
	assert.Equal(t, http.StatusNotFound, postThumbResp.StatusCode)
	postThumbResp.Body.Close()
}

// ─── Full Flow with Image ────────────────────────────────────────────────────

func TestE2E_PieceDefinition_ImageFullFlow(t *testing.T) {
	clean(t)
	_, token := createUserForClients(t, "Ana", "ana@example.com")

	// 1. Create with image
	createResp := helpers.AuthPostMultipart(t, pieceDefURL(), token, map[string]any{
		"name":             "Rectangle",
		"dimension_schema": []string{"Length", "Width"},
	}, helpers.MinimalPNG(), "rect.png")
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := helpers.ParseBody(t, createResp)
	defID := created["id"].(string)
	assert.NotEmpty(t, created["image_url"])
	assert.NotEmpty(t, created["thumbnail_url"])

	// 2. Verify image is accessible via file proxy
	imgResp := helpers.GetJSON(t, rootURL(created["image_url"].(string)))
	assert.Equal(t, http.StatusOK, imgResp.StatusCode)
	imgResp.Body.Close()

	// 3. Update with new image
	updateResp := helpers.AuthPutMultipart(t, pieceDefURL(defID), token, map[string]any{
		"name": "Updated Rectangle",
	}, helpers.MinimalPNG(), "new-rect.png")
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	updated := helpers.ParseBody(t, updateResp)
	assert.Equal(t, "Updated Rectangle", updated["name"])
	assert.NotEqual(t, created["image_url"], updated["image_url"], "image should be replaced")

	// 4. Old image should be gone
	oldImgResp := helpers.GetJSON(t, rootURL(created["image_url"].(string)))
	assert.Equal(t, http.StatusNotFound, oldImgResp.StatusCode)
	oldImgResp.Body.Close()

	// 5. New image should be accessible
	newImgResp := helpers.GetJSON(t, rootURL(updated["image_url"].(string)))
	assert.Equal(t, http.StatusOK, newImgResp.StatusCode)
	newImgResp.Body.Close()

	// 6. Delete
	delResp := helpers.AuthDeleteJSON(t, pieceDefURL(defID), token)
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)

	// 7. Files should be cleaned up
	finalResp := helpers.GetJSON(t, rootURL(updated["image_url"].(string)))
	assert.Equal(t, http.StatusNotFound, finalResp.StatusCode)
	finalResp.Body.Close()
}
