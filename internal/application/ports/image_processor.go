package ports

import (
	"errors"
	"io"
)

// ErrImageDecode is returned when the image data cannot be decoded.
// Adapters wrapping third-party decoders should wrap their decode
// errors with this sentinel so the application layer can distinguish
// a corrupt/invalid image from an internal failure.
var ErrImageDecode = errors.New("image decode failed")

// ImageProcessor is the outbound port for image manipulation.
// Implementations live in infrastructure (e.g. disintegration/imaging).
type ImageProcessor interface {
	// GenerateThumbnail creates a resized thumbnail from the source image.
	// Returns a reader with the thumbnail content, its content type, and size.
	GenerateThumbnail(src io.Reader) (io.Reader, string, int64, error)
}
