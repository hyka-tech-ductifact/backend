package http_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ductifact/internal/application/ports"
	handler "ductifact/internal/infrastructure/adapters/inbound/http"
	"ductifact/test/unit/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupFileRouter(storage *mocks.MockFileStorage) *gin.Engine {
	r := gin.New()
	h := handler.NewFileHandler(storage)
	r.GET("/v1/files/*filepath", h.ServeFile)
	return r
}

func TestServeFile_WithExistingFile_StreamsContent(t *testing.T) {
	content := "fake-image-bytes"
	storage := &mocks.MockFileStorage{
		GetObjectFn: func(ctx context.Context, key string) (*ports.FileObject, error) {
			return &ports.FileObject{
				Reader:      io.NopCloser(strings.NewReader(content)),
				ContentType: "image/png",
				Size:        int64(len(content)),
			}, nil
		},
	}

	router := setupFileRouter(storage)
	req := httptest.NewRequest(http.MethodGet, "/v1/files/piece-definitions/abc/original.png", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.Equal(t, "16", w.Header().Get("Content-Length"))
	assert.Contains(t, w.Header().Get("Cache-Control"), "immutable")
	assert.Equal(t, content, w.Body.String())
}

func TestServeFile_WithMissingFile_Returns404(t *testing.T) {
	storage := &mocks.MockFileStorage{
		GetObjectFn: func(ctx context.Context, key string) (*ports.FileObject, error) {
			return nil, ports.ErrFileNotFound
		},
	}

	router := setupFileRouter(storage)
	req := httptest.NewRequest(http.MethodGet, "/v1/files/nonexistent/key.png", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "file not found")
}

func TestServeFile_WhenStorageFails_Returns500(t *testing.T) {
	storage := &mocks.MockFileStorage{
		GetObjectFn: func(ctx context.Context, key string) (*ports.FileObject, error) {
			return nil, errors.New("storage unavailable")
		},
	}

	router := setupFileRouter(storage)
	req := httptest.NewRequest(http.MethodGet, "/v1/files/some/key.png", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "internal server error")
}

func TestServeFile_ReceivesCorrectKeyWithoutLeadingSlash(t *testing.T) {
	var receivedKey string
	storage := &mocks.MockFileStorage{
		GetObjectFn: func(ctx context.Context, key string) (*ports.FileObject, error) {
			receivedKey = key
			return &ports.FileObject{
				Reader:      io.NopCloser(strings.NewReader("")),
				ContentType: "image/jpeg",
				Size:        0,
			}, nil
		},
	}

	router := setupFileRouter(storage)
	req := httptest.NewRequest(http.MethodGet, "/v1/files/piece-definitions/uuid/thumb.jpg", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "piece-definitions/uuid/thumb.jpg", receivedKey)
}

func TestServeFile_WithWhitespaceOnlyPath_Returns400(t *testing.T) {
	router := setupFileRouter(&mocks.MockFileStorage{})
	req := httptest.NewRequest(http.MethodGet, "/v1/files/%C2%A0", nil) // non-breaking space
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "file path is required")
}

func TestServeFile_WithNoSlashInPath_Returns400(t *testing.T) {
	router := setupFileRouter(&mocks.MockFileStorage{})
	req := httptest.NewRequest(http.MethodGet, "/v1/files/invalid-no-slash", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "file path is required")
}
