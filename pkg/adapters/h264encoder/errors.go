package h264encoder

import "errors"

var (
	// ErrNotInitialized is returned when encoder methods are called before initialization.
	ErrNotInitialized = errors.New("h264encoder: encoder not initialized")

	// ErrEncodingFailed is returned when encoding a frame fails.
	ErrEncodingFailed = errors.New("h264encoder: encoding failed")

	// ErrNoFrames is returned when trying to build MP4 with no frames.
	ErrNoFrames = errors.New("h264encoder: no frames to encode")

	// ErrFFmpegNotFound is returned when ffmpeg is not found in PATH (Linux only).
	ErrFFmpegNotFound = errors.New("h264encoder: ffmpeg not found in PATH")

	// ErrPlatformNotSupported is returned when the platform doesn't support H.264 encoding.
	ErrPlatformNotSupported = errors.New("h264encoder: platform not supported")
)
