// Package h264encoder provides H.264 video encoding using platform-native APIs.
// - macOS: VideoToolbox
// - Windows: Media Foundation
// - Linux: ffmpeg (external process)
package h264encoder

import (
	"image"
	"sync"

	"github.com/user/loadshow/pkg/ports"
)

// customFFmpegPath is set via SetFFmpegPath for Linux.
var customFFmpegPath string

// SetFFmpegPath sets a custom ffmpeg path for Linux.
// This must be called before IsAvailable() or New().
func SetFFmpegPath(path string) {
	customFFmpegPath = path
}

// GetFFmpegPath returns the current custom ffmpeg path.
func GetFFmpegPath() string {
	return customFFmpegPath
}

// IsAvailable checks if H.264 encoding is available on this platform.
// This returns true if either native encoding or ffmpeg is available.
func IsAvailable() bool {
	return IsNativeAvailable() || IsFFmpegAvailable()
}

// IsNativeAvailable checks if native H.264 encoding is available.
// On macOS: VideoToolbox (always available)
// On Windows: Media Foundation (always available)
// On Linux: Not available (use ffmpeg instead)
func IsNativeAvailable() bool {
	return checkPlatformAvailability()
}

// EncoderType represents the type of H.264 encoder being used.
type EncoderType int

const (
	EncoderTypeNone EncoderType = iota
	EncoderTypeNative
	EncoderTypeFFmpeg
)

func (t EncoderType) String() string {
	switch t {
	case EncoderTypeNative:
		return "native"
	case EncoderTypeFFmpeg:
		return "ffmpeg"
	default:
		return "none"
	}
}

// NewBestAvailable returns the best available H.264 encoder with its type.
// Priority: Native (VideoToolbox/Media Foundation) > FFmpeg
// If no encoder is available, returns nil and EncoderTypeNone.
func NewBestAvailable() (ports.VideoEncoder, EncoderType) {
	if IsNativeAvailable() {
		return New(), EncoderTypeNative
	}
	if IsFFmpegAvailable() {
		return NewFFmpegEncoder(), EncoderTypeFFmpeg
	}
	return nil, EncoderTypeNone
}

// encodedFrame represents a single encoded H.264 frame.
type encodedFrame struct {
	data        []byte
	timestampUs int64
	isKeyframe  bool
}

// Encoder implements ports.VideoEncoder using platform-native H.264 encoding.
type Encoder struct {
	mu sync.Mutex

	width   int
	height  int
	fps     float64
	options ports.EncoderOptions

	frames     []encodedFrame
	frameCount int

	// Platform-specific encoder handle
	platformEncoder platformEncoder
}

// platformEncoder is implemented by platform-specific code.
type platformEncoder interface {
	init(width, height int, fps float64, opts ports.EncoderOptions) error
	encodeFrame(img image.Image, timestampMs int) ([]encodedFrame, error)
	flush() ([]encodedFrame, error)
	close()
}

// New creates a new H.264 encoder.
func New() *Encoder {
	return &Encoder{}
}

// Begin initializes the encoder.
func (e *Encoder) Begin(width, height int, fps float64, opts ports.EncoderOptions) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.width = width
	e.height = height
	e.fps = fps
	e.options = opts
	e.frames = nil
	e.frameCount = 0

	// Create platform-specific encoder
	e.platformEncoder = newPlatformEncoder()
	if err := e.platformEncoder.init(width, height, fps, opts); err != nil {
		return err
	}

	return nil
}

// EncodeFrame encodes a single frame.
func (e *Encoder) EncodeFrame(img image.Image, timestampMs int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.platformEncoder == nil {
		return ErrNotInitialized
	}

	frames, err := e.platformEncoder.encodeFrame(img, timestampMs)
	if err != nil {
		return err
	}

	e.frames = append(e.frames, frames...)
	e.frameCount++
	return nil
}

// End finalizes encoding and returns the MP4 data.
func (e *Encoder) End() ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.platformEncoder == nil {
		return nil, ErrNotInitialized
	}

	// Flush remaining frames
	frames, err := e.platformEncoder.flush()
	if err != nil {
		return nil, err
	}
	e.frames = append(e.frames, frames...)

	var mp4Data []byte

	// Check if the platform encoder provides complete MP4 output (e.g., ffmpeg on Linux)
	if provider, ok := e.platformEncoder.(mp4Provider); ok {
		mp4Data, err = provider.getOutputMP4()
		if err != nil {
			// Fallback to building MP4 ourselves
			mp4Data, err = e.buildMP4()
			if err != nil {
				return nil, err
			}
		}
	} else {
		// Build MP4 container from raw frames
		mp4Data, err = e.buildMP4()
		if err != nil {
			return nil, err
		}
	}

	// Cleanup
	e.platformEncoder.close()
	e.platformEncoder = nil

	return mp4Data, nil
}

// mp4Provider is implemented by encoders that produce complete MP4 output directly.
type mp4Provider interface {
	getOutputMP4() ([]byte, error)
}

// Ensure Encoder implements ports.VideoEncoder
var _ ports.VideoEncoder = (*Encoder)(nil)
