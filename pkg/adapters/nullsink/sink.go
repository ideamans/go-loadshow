// Package nullsink provides a no-op debug sink implementation.
package nullsink

import (
	"image"

	"github.com/user/loadshow/pkg/ports"
)

// Sink is a no-op implementation of ports.DebugSink.
// It discards all debug output.
type Sink struct{}

// New creates a new NullSink.
func New() *Sink {
	return &Sink{}
}

// Enabled returns false as this sink discards all output.
func (s *Sink) Enabled() bool {
	return false
}

// SaveLayoutJSON does nothing.
func (s *Sink) SaveLayoutJSON(data []byte) error {
	return nil
}

// SaveLayoutSVG does nothing.
func (s *Sink) SaveLayoutSVG(data []byte) error {
	return nil
}

// SaveRecordingJSON does nothing.
func (s *Sink) SaveRecordingJSON(data []byte) error {
	return nil
}

// SaveRawFrame does nothing.
func (s *Sink) SaveRawFrame(index int, data []byte) error {
	return nil
}

// SaveBanner does nothing.
func (s *Sink) SaveBanner(img image.Image) error {
	return nil
}

// SaveComposedFrame does nothing.
func (s *Sink) SaveComposedFrame(index int, img image.Image) error {
	return nil
}

// Ensure Sink implements ports.DebugSink
var _ ports.DebugSink = (*Sink)(nil)
