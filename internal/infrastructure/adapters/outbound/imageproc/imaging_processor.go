package imageproc

import (
	"bytes"
	"fmt"
	"io"

	"ductifact/internal/application/ports"

	"github.com/disintegration/imaging"
)

const (
	thumbWidth  = 200
	thumbHeight = 150
)

// ImagingProcessor implements ports.ImageProcessor using disintegration/imaging.
// Pure Go — no CGo dependencies.
type ImagingProcessor struct{}

// NewImagingProcessor creates a new ImagingProcessor.
func NewImagingProcessor() *ImagingProcessor {
	return &ImagingProcessor{}
}

// GenerateThumbnail creates a 200×150 JPEG thumbnail from the source image.
// Uses Lanczos resampling for quality. Always outputs JPEG regardless of input format.
func (p *ImagingProcessor) GenerateThumbnail(src io.Reader) (io.Reader, string, int64, error) {
	img, err := imaging.Decode(src)
	if err != nil {
		return nil, "", 0, fmt.Errorf("decoding image: %w", fmt.Errorf("%s: %w", err.Error(), ports.ErrImageDecode))
	}

	thumb := imaging.Thumbnail(img, thumbWidth, thumbHeight, imaging.Lanczos)

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, thumb, imaging.JPEG, imaging.JPEGQuality(80)); err != nil {
		return nil, "", 0, fmt.Errorf("encoding thumbnail: %w", err)
	}

	return bytes.NewReader(buf.Bytes()), "image/jpeg", int64(buf.Len()), nil
}
