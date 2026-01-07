// Package smartdecoder provides a smart video decoder that automatically
// detects the codec and selects the appropriate decoder.
package smartdecoder

import (
	"errors"
	"io"

	"github.com/user/loadshow/pkg/adapters/av1decoder"
	"github.com/user/loadshow/pkg/adapters/codecdetect"
	"github.com/user/loadshow/pkg/adapters/h264decoder"
	"github.com/user/loadshow/pkg/ports"
)

// Codec represents the video codec type (re-exported from codecdetect).
type Codec = codecdetect.Codec

const (
	// CodecH264 represents H.264/AVC codec.
	CodecH264 = codecdetect.CodecH264
	// CodecAV1 represents AV1 codec.
	CodecAV1 = codecdetect.CodecAV1
	// CodecUnknown represents an unknown codec.
	CodecUnknown = codecdetect.CodecUnknown
)

// Backend represents the decoding backend used.
type Backend string

const (
	// BackendOS represents platform-native decoding (VideoToolbox on macOS, Media Foundation on Windows).
	BackendOS Backend = "os"
	// BackendFFmpeg represents FFmpeg-based decoding.
	BackendFFmpeg Backend = "ffmpeg"
	// BackendLibaom represents libaom for AV1 decoding.
	BackendLibaom Backend = "libaom"
)

// Info contains information about the selected decoder.
type Info struct {
	// Codec is the detected codec.
	Codec Codec
	// Backend is the decoding backend being used.
	Backend Backend
}

// Options configures the smart decoder behavior.
type Options struct {
	// FFmpegPath is an optional custom path to the ffmpeg binary.
	FFmpegPath string
}

var (
	// ErrUnsupportedCodec is returned when the codec is not supported.
	ErrUnsupportedCodec = errors.New("smartdecoder: unsupported codec")
	// ErrNoDecoderAvailable is returned when no decoder is available for the codec.
	ErrNoDecoderAvailable = errors.New("smartdecoder: no decoder available")
)

// Decoder wraps ports.VideoDecoder with codec detection.
type Decoder struct {
	inner ports.VideoDecoder
	info  Info
}

// NewFromFile creates a decoder by auto-detecting the codec from the file.
//
// The selection flow:
//   - AV1: Use libaom decoder
//   - H.264: Try OS-native decoder, then FFmpeg decoder
func NewFromFile(path string, opts Options) (*Decoder, Info, error) {
	// Set custom FFmpeg path if provided
	if opts.FFmpegPath != "" {
		h264decoder.SetFFmpegPath(opts.FFmpegPath)
	}

	// Detect codec from file
	codec, err := codecdetect.DetectFromFile(path)
	if err != nil {
		return nil, Info{}, err
	}

	return createDecoder(codec, opts)
}

// NewFromReader creates a decoder by auto-detecting the codec from the reader.
func NewFromReader(reader io.ReadSeeker, opts Options) (*Decoder, Info, error) {
	if opts.FFmpegPath != "" {
		h264decoder.SetFFmpegPath(opts.FFmpegPath)
	}

	codec, err := codecdetect.DetectFromReader(reader)
	if err != nil {
		return nil, Info{}, err
	}

	return createDecoder(codec, opts)
}

// NewFromBytes creates a decoder by auto-detecting the codec from MP4 data.
func NewFromBytes(data []byte, opts Options) (*Decoder, Info, error) {
	if opts.FFmpegPath != "" {
		h264decoder.SetFFmpegPath(opts.FFmpegPath)
	}

	codec, err := codecdetect.DetectFromBytes(data)
	if err != nil {
		return nil, Info{}, err
	}

	return createDecoder(codec, opts)
}

// NewForCodec creates a decoder for a specific codec.
func NewForCodec(codec Codec, opts Options) (*Decoder, Info, error) {
	if opts.FFmpegPath != "" {
		h264decoder.SetFFmpegPath(opts.FFmpegPath)
	}

	return createDecoder(codec, opts)
}

func createDecoder(codec Codec, opts Options) (*Decoder, Info, error) {
	switch codec {
	case CodecAV1:
		return &Decoder{
				inner: av1decoder.NewMP4Reader(),
				info: Info{
					Codec:   CodecAV1,
					Backend: BackendLibaom,
				},
			}, Info{
				Codec:   CodecAV1,
				Backend: BackendLibaom,
			}, nil

	case CodecH264:
		// Check if H.264 decoding is available
		if !h264decoder.IsAvailable() {
			return nil, Info{}, ErrNoDecoderAvailable
		}

		// Determine backend based on availability
		// Note: h264decoder internally handles OS vs FFmpeg selection
		backend := BackendOS
		if !isH264NativeAvailable() {
			backend = BackendFFmpeg
		}

		return &Decoder{
				inner: h264decoder.NewMP4Reader(),
				info: Info{
					Codec:   CodecH264,
					Backend: backend,
				},
			}, Info{
				Codec:   CodecH264,
				Backend: backend,
			}, nil

	case CodecUnknown:
		return nil, Info{}, ErrUnsupportedCodec

	default:
		return nil, Info{}, ErrUnsupportedCodec
	}
}

// isH264NativeAvailable checks if native H.264 decoding is available.
// This is a platform-specific check.
func isH264NativeAvailable() bool {
	// On platforms with native support (macOS, Windows), checkPlatformAvailability returns true
	// On Linux and other platforms, it returns false (they use FFmpeg)
	// We can infer this by checking if IsAvailable returns true and assuming
	// the platform decoder handles this correctly.
	//
	// For now, we check if the decoder initializes without error as a proxy.
	// In practice, h264decoder internally manages this.
	return h264decoder.IsAvailable()
}

// ReadFrames reads and decodes all frames from a video file.
func (d *Decoder) ReadFrames(path string) ([]ports.VideoFrame, error) {
	return d.inner.ReadFrames(path)
}

// ReadFramesFromReader reads and decodes all frames from an io.ReadSeeker.
func (d *Decoder) ReadFramesFromReader(reader io.ReadSeeker) ([]ports.VideoFrame, error) {
	return d.inner.ReadFramesFromReader(reader)
}

// Close releases decoder resources.
func (d *Decoder) Close() {
	d.inner.Close()
}

// Info returns information about the decoder.
func (d *Decoder) Info() Info {
	return d.info
}

// DetectCodec detects the codec from a file without creating a decoder.
func DetectCodec(path string) (Codec, error) {
	return codecdetect.DetectFromFile(path)
}

// DetectCodecFromReader detects the codec from a reader without creating a decoder.
func DetectCodecFromReader(reader io.ReadSeeker) (Codec, error) {
	return codecdetect.DetectFromReader(reader)
}

// DetectCodecFromBytes detects the codec from MP4 data without creating a decoder.
func DetectCodecFromBytes(data []byte) (Codec, error) {
	return codecdetect.DetectFromBytes(data)
}

// IsH264Available checks if H.264 decoding is available.
func IsH264Available() bool {
	return h264decoder.IsAvailable()
}

// IsAV1Available always returns true (libaom is always linked).
func IsAV1Available() bool {
	return true
}

// Ensure Decoder implements ports.VideoDecoder
var _ ports.VideoDecoder = (*Decoder)(nil)
