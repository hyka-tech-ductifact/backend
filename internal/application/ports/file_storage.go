package ports

import (
	"context"
	"io"
)

// FileStorage is the outbound port for storing and retrieving files.
// The implementation (MinIO, S3, local disk, etc.) lives in infrastructure.
type FileStorage interface {
	// Upload stores a file and returns the object key (relative path).
	// The caller provides the key, a reader with the file content, the
	// content type (e.g. "image/png"), and the size in bytes.
	Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error

	// Delete removes a file by its object key.
	// Returns nil if the file doesn't exist (idempotent).
	Delete(ctx context.Context, key string) error
}
