package mocks

import (
	"image"

	"github.com/user/loadshow/pkg/ports"
)

// VideoEncoder is a mock implementation of ports.VideoEncoder.
type VideoEncoder struct {
	BeginFunc       func(width, height int, fps float64, opts ports.EncoderOptions) error
	EncodeFrameFunc func(img image.Image, timestampMs int) error
	EndFunc         func() ([]byte, error)

	// Recorded calls for verification
	BeginCalled      bool
	EncodeFrameCalls []EncodeFrameCall
	EndCalled        bool
}

// EncodeFrameCall records a call to EncodeFrame.
type EncodeFrameCall struct {
	TimestampMs int
}

func (m *VideoEncoder) Begin(width, height int, fps float64, opts ports.EncoderOptions) error {
	m.BeginCalled = true
	if m.BeginFunc != nil {
		return m.BeginFunc(width, height, fps, opts)
	}
	return nil
}

func (m *VideoEncoder) EncodeFrame(img image.Image, timestampMs int) error {
	m.EncodeFrameCalls = append(m.EncodeFrameCalls, EncodeFrameCall{TimestampMs: timestampMs})
	if m.EncodeFrameFunc != nil {
		return m.EncodeFrameFunc(img, timestampMs)
	}
	return nil
}

func (m *VideoEncoder) End() ([]byte, error) {
	m.EndCalled = true
	if m.EndFunc != nil {
		return m.EndFunc()
	}
	// Return minimal WebM header
	return []byte{0x1A, 0x45, 0xDF, 0xA3}, nil
}

var _ ports.VideoEncoder = (*VideoEncoder)(nil)
