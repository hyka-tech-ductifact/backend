package mocks

import (
	"context"
	"io"
)

// MockFileStorage implements ports.FileStorage for testing.
type MockFileStorage struct {
	UploadFn func(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error
	DeleteFn func(ctx context.Context, key string) error
}

func (m *MockFileStorage) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error {
	if m.UploadFn != nil {
		return m.UploadFn(ctx, key, reader, contentType, size)
	}
	return nil
}

func (m *MockFileStorage) Delete(ctx context.Context, key string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, key)
	}
	return nil
}
