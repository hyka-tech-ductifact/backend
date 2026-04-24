"""
Schemathesis hooks — custom media type strategies for image uploads.

Our piece-definition endpoints accept multipart/form-data with an optional
`image` field encoded as image/png, image/jpeg, or image/webp.  Schemathesis
doesn't know how to generate valid image bytes out of the box, so we register
minimal valid specimens for each content type.

Ref: https://schemathesis.readthedocs.io/en/stable/guides/custom-media-types/
"""

from hypothesis import strategies as st
import schemathesis

# ── Minimal valid images (1×1 pixel) ─────────────────────────

# Minimal valid PNG — 1×1 red pixel, RGBA
_MINIMAL_PNG = (
    b"\x89PNG\r\n\x1a\n"  # PNG signature
    b"\x00\x00\x00\rIHDR"  # IHDR chunk
    b"\x00\x00\x00\x01"  # width  = 1
    b"\x00\x00\x00\x01"  # height = 1
    b"\x08\x02"  # bit depth 8, color type 2 (RGB)
    b"\x00\x00\x00"  # compression, filter, interlace
    b"\x90wS\xde"  # CRC
    b"\x00\x00\x00\x0cIDATx"  # IDAT chunk
    b"\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05"
    b"\x18\xd8N"  # CRC
    b"\x00\x00\x00\x00IEND\xaeB`\x82"  # IEND chunk
)

# Minimal valid JPEG — 1×1 pixel
_MINIMAL_JPEG = (
    b"\xff\xd8\xff\xe0"  # SOI + APP0 marker
    b"\x00\x10JFIF\x00\x01\x01\x01\x00H\x00H\x00\x00"  # JFIF header
    b"\xff\xdb\x00C\x00"  # DQT marker
    b"\x08\x06\x06\x07\x06\x05\x08\x07\x07\x07\t\t"
    b"\x08\n\x0c\x14\r\x0c\x0b\x0b\x0c\x19\x12\x13"
    b"\x0f\x14\x1d\x1a\x1f\x1e\x1d\x1a\x1c\x1c $.\""
    b" ,#\x1c\x1c(7),01444\x1f'9=82<.342"
    b"\xff\xc0\x00\x11\x08"  # SOF0 marker
    b"\x00\x01\x00\x01"  # height=1, width=1
    b"\x01\x01\x11\x00"  # 1 component, sampling, quant table
    # Minimal Huffman + scan
    b"\xff\xc4\x00\x14\x00\x01\x00\x00\x00\x00\x00"
    b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
    b"\xff\xda\x00\x08\x01\x01\x00\x00?\x00T\xdb"
    b"\x9e\x1f\xff\xd9"  # EOI
)

# Minimal valid WebP — 1×1 pixel (VP8 lossy)
_MINIMAL_WEBP = (
    b"RIFF"  # RIFF header
    b"\x24\x00\x00\x00"  # file size - 8
    b"WEBP"  # WEBP signature
    b"VP8 "  # VP8 chunk
    b"\x18\x00\x00\x00"  # chunk size
    b"\x30\x01\x00\x9d\x01\x2a"  # VP8 bitstream header
    b"\x01\x00\x01\x00"  # width=1, height=1
    b"\x01\x40\x25\xa4\x00\x03\x70\x00\xfe\xfb"
    b"\x94\x00\x00"  # pixel data
)

# Strategy: randomly pick one of the three valid image formats
_image_strategy = st.sampled_from([_MINIMAL_PNG, _MINIMAL_JPEG, _MINIMAL_WEBP])

# Register for all image/* media types (covers image/png, image/jpeg, image/webp)
schemathesis.openapi.media_type("image/*", _image_strategy)
