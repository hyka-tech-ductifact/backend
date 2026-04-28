package helpers

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// PostJSON sends a POST request with a JSON body and returns the response.
func PostJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	return resp
}

// PutJSON sends a PUT request with a JSON body and returns the response.
func PutJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// DeleteJSON sends a DELETE request and returns the response.
func DeleteJSON(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// GetJSON sends a GET request and returns the response.
func GetJSON(t *testing.T, url string) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	require.NoError(t, err)
	return resp
}

// --- Authenticated versions (with Bearer token) ---

// AuthGetJSON sends GET with Authorization: Bearer <token>.
func AuthGetJSON(t *testing.T, url, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// AuthPostJSON sends POST with Authorization: Bearer <token>.
func AuthPostJSON(t *testing.T, url, token string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// AuthPutJSON sends PUT with Authorization: Bearer <token>.
func AuthPutJSON(t *testing.T, url, token string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// AuthDeleteJSON sends DELETE with Authorization: Bearer <token>.
func AuthDeleteJSON(t *testing.T, url, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// ParseBody decodes the response body as JSON into a map.
// It closes the response body after reading.
func ParseBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var body map[string]any
	err := json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	return body
}

// ParseBodyArray decodes the response body as a JSON array into the provided slice.
// It closes the response body after reading.
func ParseBodyArray(t *testing.T, resp *http.Response, target any) {
	t.Helper()
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(target)
	require.NoError(t, err)
}

// --- Multipart helpers (for multipart/form-data endpoints) ---

// MinimalPNG returns a valid 1×1 pixel PNG image suitable for testing uploads.
func MinimalPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// buildMultipart creates a multipart/form-data body with a "data" JSON field
// and an optional "image" file part. Returns the body buffer and the Content-Type
// header value (including multipart boundary).
func buildMultipart(t *testing.T, data any, imageBytes []byte, imageFilename string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if data != nil {
		jsonBytes, err := json.Marshal(data)
		require.NoError(t, err)
		require.NoError(t, writer.WriteField("data", string(jsonBytes)))
	}

	if imageBytes != nil {
		part, err := writer.CreateFormFile("image", imageFilename)
		require.NoError(t, err)
		_, err = part.Write(imageBytes)
		require.NoError(t, err)
	}

	require.NoError(t, writer.Close())
	return &body, writer.FormDataContentType()
}

// PostMultipart sends a POST multipart/form-data request (no auth).
func PostMultipart(t *testing.T, url string, data any, imageBytes []byte, imageFilename string) *http.Response {
	t.Helper()
	body, contentType := buildMultipart(t, data, imageBytes, imageFilename)
	req, err := http.NewRequest(http.MethodPost, url, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// AuthPostMultipart sends a POST multipart/form-data request with Authorization: Bearer <token>.
func AuthPostMultipart(
	t *testing.T,
	url, token string,
	data any,
	imageBytes []byte,
	imageFilename string,
) *http.Response {
	t.Helper()
	body, contentType := buildMultipart(t, data, imageBytes, imageFilename)
	req, err := http.NewRequest(http.MethodPost, url, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// AuthPutMultipart sends a PUT multipart/form-data request with Authorization: Bearer <token>.
func AuthPutMultipart(
	t *testing.T,
	url, token string,
	data any,
	imageBytes []byte,
	imageFilename string,
) *http.Response {
	t.Helper()
	body, contentType := buildMultipart(t, data, imageBytes, imageFilename)
	req, err := http.NewRequest(http.MethodPut, url, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}
