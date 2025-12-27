// Package smartencoder provides a smart video encoder that automatically
// selects the best available codec with fallback support.
package smartencoder

import (
	"errors"

	"github.com/user/loadshow/pkg/adapters/av1encoder"
	"github.com/user/loadshow/pkg/adapters/h264encoder"
	"github.com/user/loadshow/pkg/ports"
)

// Codec represents the video codec type.
type Codec string

const (
	// CodecH264 represents H.264/AVC codec.
	CodecH264 Codec = "h264"
	// CodecAV1 represents AV1 codec.
	CodecAV1 Codec = "av1"
)

// Backend represents the encoding backend used.
type Backend string

const (
	// BackendOS represents platform-native encoding (VideoToolbox on macOS, Media Foundation on Windows).
	BackendOS Backend = "os"
	// BackendFFmpeg represents FFmpeg-based encoding.
	BackendFFmpeg Backend = "ffmpeg"
	// BackendLibaom represents libaom for AV1 encoding.
	BackendLibaom Backend = "libaom"
)

// Info contains information about the selected encoder.
type Info struct {
	// Codec is the actual codec being used.
	Codec Codec
	// Backend is the encoding backend being used.
	Backend Backend
	// RequestedCodec is the codec that was originally requested.
	RequestedCodec Codec
	// FallbackUsed indicates whether a fallback occurred.
	FallbackUsed bool
}

// Options configures the smart encoder behavior.
type Options struct {
	// FFmpegPath is an optional custom path to the ffmpeg binary.
	FFmpegPath string
	// AllowFallback enables fallback to AV1 when H.264 is not available.
	// Defaults to true.
	AllowFallback bool
	// Logger is used to log fallback warnings.
	Logger ports.Logger
}

var (
	// ErrNoEncoderAvailable is returned when no encoder is available.
	ErrNoEncoderAvailable = errors.New("smartencoder: no encoder available")
)

// New creates a new video encoder with automatic codec selection.
//
// The selection flow for H.264:
//  1. Try OS-native encoder (VideoToolbox on macOS, Media Foundation on Windows)
//  2. Try FFmpeg encoder
//  3. If AllowFallback is true, fall back to AV1 (libaom)
//
// For AV1, libaom is always used.
func New(preferred Codec, opts Options) (ports.VideoEncoder, Info, error) {
	// Set default for AllowFallback
	if !opts.AllowFallback {
		// Check if caller explicitly set it to false or just didn't set it
		// Default to true for better UX
		opts.AllowFallback = true
	}

	// Set custom FFmpeg path if provided
	if opts.FFmpegPath != "" {
		h264encoder.SetFFmpegPath(opts.FFmpegPath)
	}

	info := Info{
		RequestedCodec: preferred,
	}

	switch preferred {
	case CodecH264:
		return selectH264Encoder(opts, info)
	case CodecAV1:
		return av1encoder.New(), Info{
			Codec:          CodecAV1,
			Backend:        BackendLibaom,
			RequestedCodec: CodecAV1,
			FallbackUsed:   false,
		}, nil
	default:
		// Default to H.264 selection
		return selectH264Encoder(opts, info)
	}
}

// NewWithoutFallback creates a new encoder without fallback capability.
// Returns an error if the requested codec is not available.
func NewWithoutFallback(preferred Codec, opts Options) (ports.VideoEncoder, Info, error) {
	opts.AllowFallback = false
	return newInternal(preferred, opts)
}

func newInternal(preferred Codec, opts Options) (ports.VideoEncoder, Info, error) {
	if opts.FFmpegPath != "" {
		h264encoder.SetFFmpegPath(opts.FFmpegPath)
	}

	info := Info{
		RequestedCodec: preferred,
	}

	switch preferred {
	case CodecH264:
		return selectH264Encoder(opts, info)
	case CodecAV1:
		return av1encoder.New(), Info{
			Codec:          CodecAV1,
			Backend:        BackendLibaom,
			RequestedCodec: CodecAV1,
			FallbackUsed:   false,
		}, nil
	default:
		return selectH264Encoder(opts, info)
	}
}

func selectH264Encoder(opts Options, info Info) (ports.VideoEncoder, Info, error) {
	// Try OS-native encoder first
	if h264encoder.IsNativeAvailable() {
		return h264encoder.New(), Info{
			Codec:          CodecH264,
			Backend:        BackendOS,
			RequestedCodec: info.RequestedCodec,
			FallbackUsed:   false,
		}, nil
	}

	// Try FFmpeg encoder
	if h264encoder.IsFFmpegAvailable() {
		return h264encoder.NewFFmpegEncoder(), Info{
			Codec:          CodecH264,
			Backend:        BackendFFmpeg,
			RequestedCodec: info.RequestedCodec,
			FallbackUsed:   false,
		}, nil
	}

	// H.264 not available, check if fallback is allowed
	if !opts.AllowFallback {
		return nil, Info{}, ErrNoEncoderAvailable
	}

	// Log fallback warning
	if opts.Logger != nil {
		opts.Logger.Warn("H.264 encoder not available, falling back to AV1")
	}

	// Fall back to AV1
	return av1encoder.New(), Info{
		Codec:          CodecAV1,
		Backend:        BackendLibaom,
		RequestedCodec: info.RequestedCodec,
		FallbackUsed:   true,
	}, nil
}

// IsH264Available checks if H.264 encoding is available (native or FFmpeg).
func IsH264Available() bool {
	return h264encoder.IsAvailable()
}

// IsH264NativeAvailable checks if native H.264 encoding is available.
// Returns true on macOS (VideoToolbox) and Windows (Media Foundation).
// Returns false on Linux and other platforms.
func IsH264NativeAvailable() bool {
	return h264encoder.IsNativeAvailable()
}

// IsH264FFmpegAvailable checks if FFmpeg-based H.264 encoding is available.
func IsH264FFmpegAvailable() bool {
	return h264encoder.IsFFmpegAvailable()
}

// IsAV1Available always returns true (libaom is always linked).
func IsAV1Available() bool {
	return true
}
