package ports

import (
	"io"
)

// ImageProcessor is the outbound port for image manipulation.
// Implementations live in infrastructure (e.g. disintegration/imaging).
type ImageProcessor interface {
	// GenerateThumbnail creates a resized thumbnail from the source image.
	// Returns a reader with the thumbnail content, its content type, and size.
	GenerateThumbnail(src io.Reader) (io.Reader, string, int64, error)
}
