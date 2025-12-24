// Package filesink provides a file-based debug sink implementation.
package filesink

import (
	"fmt"
	"image"
	"path/filepath"

	"github.com/user/loadshow/pkg/ports"
)

// Sink saves debug output to files.
type Sink struct {
	baseDir  string
	fs       ports.FileSystem
	renderer ports.Renderer
}

// New creates a new FileSink.
func New(baseDir string, fs ports.FileSystem, renderer ports.Renderer) *Sink {
	return &Sink{
		baseDir:  baseDir,
		fs:       fs,
		renderer: renderer,
	}
}

// Enabled returns true as this sink saves output.
func (s *Sink) Enabled() bool {
	return true
}

// SaveLayoutJSON saves the layout calculation result as JSON.
func (s *Sink) SaveLayoutJSON(data []byte) error {
	path := filepath.Join(s.baseDir, "layout.json")
	return s.fs.WriteFile(path, data)
}

// SaveLayoutSVG saves the layout visualization as SVG.
func (s *Sink) SaveLayoutSVG(data []byte) error {
	path := filepath.Join(s.baseDir, "layout.svg")
	return s.fs.WriteFile(path, data)
}

// SaveRecordingJSON saves the recording metadata as JSON.
func (s *Sink) SaveRecordingJSON(data []byte) error {
	path := filepath.Join(s.baseDir, "recording.json")
	return s.fs.WriteFile(path, data)
}

// SaveRawFrame saves a raw recording frame.
func (s *Sink) SaveRawFrame(index int, data []byte) error {
	dir := filepath.Join(s.baseDir, "frames", "raw")
	if err := s.fs.MkdirAll(dir); err != nil {
		return err
	}
	path := filepath.Join(dir, fmt.Sprintf("frame-%04d.jpg", index))
	return s.fs.WriteFile(path, data)
}

// SaveBanner saves the generated banner image.
func (s *Sink) SaveBanner(img image.Image) error {
	data, err := s.renderer.EncodeImage(img, ports.FormatPNG, 0)
	if err != nil {
		return fmt.Errorf("encode banner: %w", err)
	}
	path := filepath.Join(s.baseDir, "banner.png")
	return s.fs.WriteFile(path, data)
}

// SaveComposedFrame saves a composed frame.
func (s *Sink) SaveComposedFrame(index int, img image.Image) error {
	dir := filepath.Join(s.baseDir, "frames", "composed")
	if err := s.fs.MkdirAll(dir); err != nil {
		return err
	}
	data, err := s.renderer.EncodeImage(img, ports.FormatPNG, 0)
	if err != nil {
		return fmt.Errorf("encode composed frame: %w", err)
	}
	path := filepath.Join(dir, fmt.Sprintf("frame-%04d.png", index))
	return s.fs.WriteFile(path, data)
}

// Ensure Sink implements ports.DebugSink
var _ ports.DebugSink = (*Sink)(nil)
