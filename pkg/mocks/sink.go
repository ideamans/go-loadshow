package mocks

import (
	"image"
	"sync"

	"github.com/user/loadshow/pkg/ports"
)

// DebugSink is a mock implementation of ports.DebugSink.
type DebugSink struct {
	mu sync.RWMutex

	enabled bool

	LayoutJSON     []byte
	LayoutSVG      []byte
	RecordingJSON  []byte
	RawFrames      map[int][]byte
	Banner         image.Image
	ComposedFrames map[int]image.Image
}

// NewDebugSink creates a new mock DebugSink.
func NewDebugSink(enabled bool) *DebugSink {
	return &DebugSink{
		enabled:        enabled,
		RawFrames:      make(map[int][]byte),
		ComposedFrames: make(map[int]image.Image),
	}
}

func (m *DebugSink) Enabled() bool {
	return m.enabled
}

func (m *DebugSink) SaveLayoutJSON(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LayoutJSON = data
	return nil
}

func (m *DebugSink) SaveLayoutSVG(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LayoutSVG = data
	return nil
}

func (m *DebugSink) SaveRecordingJSON(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RecordingJSON = data
	return nil
}

func (m *DebugSink) SaveRawFrame(index int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RawFrames[index] = data
	return nil
}

func (m *DebugSink) SaveBanner(img image.Image) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Banner = img
	return nil
}

func (m *DebugSink) SaveComposedFrame(index int, img image.Image) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ComposedFrames[index] = img
	return nil
}

var _ ports.DebugSink = (*DebugSink)(nil)

// NullSink is a no-op implementation of ports.DebugSink.
type NullSink struct{}

func (m *NullSink) Enabled() bool                              { return false }
func (m *NullSink) SaveLayoutJSON(data []byte) error           { return nil }
func (m *NullSink) SaveLayoutSVG(data []byte) error            { return nil }
func (m *NullSink) SaveRecordingJSON(data []byte) error        { return nil }
func (m *NullSink) SaveRawFrame(index int, data []byte) error  { return nil }
func (m *NullSink) SaveBanner(img image.Image) error           { return nil }
func (m *NullSink) SaveComposedFrame(index int, img image.Image) error { return nil }

var _ ports.DebugSink = (*NullSink)(nil)
