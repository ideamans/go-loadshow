package ports

import (
	"image"
	"io"
)

// VideoFrame represents a decoded video frame with timing information.
type VideoFrame struct {
	Image       image.Image
	TimestampMs int
	Duration    int // Duration in milliseconds
}

// VideoDecoder abstracts video decoding operations.
type VideoDecoder interface {
	// ReadFrames reads and decodes all frames from a video file.
	ReadFrames(path string) ([]VideoFrame, error)

	// ReadFramesFromReader reads and decodes all frames from an io.ReadSeeker.
	ReadFramesFromReader(reader io.ReadSeeker) ([]VideoFrame, error)

	// Close releases decoder resources.
	Close()
}
