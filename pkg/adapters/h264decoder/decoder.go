// Package h264decoder provides H.264 video decoding using platform-native APIs.
// - macOS: VideoToolbox
// - Windows: Media Foundation
// - Linux: ffmpeg (external process)
package h264decoder

import (
	"errors"
	"image"
)

var (
	// ErrNotInitialized is returned when decoder methods are called before initialization.
	ErrNotInitialized = errors.New("h264decoder: decoder not initialized")

	// ErrDecodeFailed is returned when decoding a frame fails.
	ErrDecodeFailed = errors.New("h264decoder: decode failed")

	// ErrFFmpegNotFound is returned when ffmpeg is not found in PATH (Linux only).
	ErrFFmpegNotFound = errors.New("h264decoder: ffmpeg not found in PATH")

	// ErrPlatformNotSupported is returned when the platform doesn't support H.264 decoding.
	ErrPlatformNotSupported = errors.New("h264decoder: platform not supported")
)

// Decoder decodes H.264 video frames using platform-native APIs.
type Decoder struct {
	platformDecoder platformDecoder
}

// platformDecoder is implemented by platform-specific code.
type platformDecoder interface {
	init() error
	decodeFrame(data []byte) (image.Image, error)
	close()
}

// New creates a new H.264 decoder.
func New() *Decoder {
	return &Decoder{}
}

// Init initializes the decoder.
func (d *Decoder) Init() error {
	d.platformDecoder = newPlatformDecoder()
	return d.platformDecoder.init()
}

// DecodeFrame decodes a single H.264 frame from Annex B format.
func (d *Decoder) DecodeFrame(data []byte) (image.Image, error) {
	if d.platformDecoder == nil {
		return nil, ErrNotInitialized
	}
	return d.platformDecoder.decodeFrame(data)
}

// Close releases decoder resources.
func (d *Decoder) Close() {
	if d.platformDecoder != nil {
		d.platformDecoder.close()
		d.platformDecoder = nil
	}
}
