//go:build !darwin && !windows && !linux

package h264encoder

import (
	"image"

	"github.com/user/loadshow/pkg/ports"
)

// unsupportedEncoder is a placeholder for unsupported platforms.
type unsupportedEncoder struct{}

func newPlatformEncoder() platformEncoder {
	return &unsupportedEncoder{}
}

func (e *unsupportedEncoder) init(width, height int, fps float64, opts ports.EncoderOptions) error {
	return ErrPlatformNotSupported
}

func (e *unsupportedEncoder) encodeFrame(img image.Image, timestampMs int) ([]encodedFrame, error) {
	return nil, ErrPlatformNotSupported
}

func (e *unsupportedEncoder) flush() ([]encodedFrame, error) {
	return nil, ErrPlatformNotSupported
}

func (e *unsupportedEncoder) close() {
}
