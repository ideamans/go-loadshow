package ports

import (
	"image"
)

// VideoEncoder abstracts video encoding operations.
type VideoEncoder interface {
	// Begin initializes the encoder with the specified dimensions and frame rate.
	Begin(width, height int, fps float64, opts EncoderOptions) error

	// EncodeFrame encodes a single frame at the specified timestamp.
	EncodeFrame(img image.Image, timestampMs int) error

	// End finalizes encoding and returns the video data.
	End() ([]byte, error)
}

// EncoderOptions configures video encoding parameters.
type EncoderOptions struct {
	Bitrate int // Target bitrate in kbps
	Quality int // CRF value: 0-63 (lower is higher quality)
}
