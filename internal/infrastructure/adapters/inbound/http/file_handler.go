package http

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"ductifact/internal/application/ports"

	"github.com/gin-gonic/gin"
)

// FileHandler serves files from storage as a reverse proxy.
// The client accesses /v1/files/{key} and the handler streams
// the object directly from the FileStorage port (e.g. MinIO).
type FileHandler struct {
	fileStorage ports.FileStorage
}

// NewFileHandler creates a new FileHandler.
func NewFileHandler(fileStorage ports.FileStorage) *FileHandler {
	return &FileHandler{fileStorage: fileStorage}
}

// ServeFile handles GET /v1/files/*filepath
// Streams the file from storage to the client without buffering in memory.
func (h *FileHandler) ServeFile(c *gin.Context) {
	// Gin captures the wildcard as "filepath" (with leading slash)
	key := c.Param("filepath")
	if len(key) > 0 && key[0] == '/' {
		key = key[1:] // strip leading slash
	}
	key = strings.TrimSpace(key)

	if key == "" || !strings.Contains(key, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file path is required"})
		return
	}

	obj, err := h.fileStorage.GetObject(c.Request.Context(), key)
	if err != nil {
		if err == ports.ErrFileNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	defer obj.Reader.Close()

	// Set headers for proper content delivery
	c.Header("Content-Type", obj.ContentType)
	c.Header("Content-Length", strconv.FormatInt(obj.Size, 10))
	c.Header("Cache-Control", "public, max-age=31536000, immutable") // 1 year — keys contain UUID so they're unique

	c.Status(http.StatusOK)

	// Stream directly from storage → client (only a few KB in memory at a time)
	io.Copy(c.Writer, obj.Reader)
}
