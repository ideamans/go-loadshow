//go:build !darwin

package h264decoder

import "image"

// stubDecoder is a placeholder for non-macOS platforms.
// Windows and Linux implementations would use Media Foundation and ffmpeg respectively.
type stubDecoder struct{}

func newPlatformDecoder() platformDecoder {
	return &stubDecoder{}
}

func (d *stubDecoder) init() error {
	return ErrPlatformNotSupported
}

func (d *stubDecoder) decodeFrame(data []byte) (image.Image, error) {
	return nil, ErrPlatformNotSupported
}

func (d *stubDecoder) close() {}
