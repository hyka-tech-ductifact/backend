package mocks

import (
	"bytes"
	"io"
)

// MockImageProcessor implements ports.ImageProcessor for testing.
type MockImageProcessor struct {
	GenerateThumbnailFn func(src io.Reader) (io.Reader, string, int64, error)
}

func (m *MockImageProcessor) GenerateThumbnail(src io.Reader) (io.Reader, string, int64, error) {
	if m.GenerateThumbnailFn != nil {
		return m.GenerateThumbnailFn(src)
	}
	// Return a small fake thumbnail by default
	data := []byte("fake-thumbnail")
	return bytes.NewReader(data), "image/jpeg", int64(len(data)), nil
}
