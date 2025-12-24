// Package encode implements the video encoding stage.
package encode

import (
	"context"
	"fmt"

	"github.com/user/loadshow/pkg/pipeline"
	"github.com/user/loadshow/pkg/ports"
)

// Stage encodes composed frames into a WebM video.
type Stage struct {
	encoder ports.VideoEncoder
}

// NewStage creates a new encode stage.
func NewStage(encoder ports.VideoEncoder) *Stage {
	return &Stage{
		encoder: encoder,
	}
}

// Execute encodes all frames into a video.
func (s *Stage) Execute(ctx context.Context, input pipeline.EncodeInput) (pipeline.EncodeResult, error) {
	result := pipeline.EncodeResult{}

	if len(input.Frames) == 0 {
		return result, fmt.Errorf("no frames to encode")
	}

	// Get dimensions from first frame
	firstFrame := input.Frames[0].Image
	bounds := firstFrame.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Initialize encoder
	opts := ports.EncoderOptions{
		Bitrate: input.Bitrate,
		Quality: input.Quality,
	}

	if err := s.encoder.Begin(width, height, input.FPS, opts); err != nil {
		return result, fmt.Errorf("begin encoding: %w", err)
	}

	// Encode each frame
	for _, frame := range input.Frames {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		if err := s.encoder.EncodeFrame(frame.Image, frame.TimestampMs); err != nil {
			return result, fmt.Errorf("encode frame at %dms: %w", frame.TimestampMs, err)
		}
	}

	// Add outro (hold last frame)
	if input.OutroMs > 0 && len(input.Frames) > 0 {
		lastFrame := input.Frames[len(input.Frames)-1]
		outroTimestamp := lastFrame.TimestampMs + input.OutroMs
		if err := s.encoder.EncodeFrame(lastFrame.Image, outroTimestamp); err != nil {
			return result, fmt.Errorf("encode outro frame: %w", err)
		}
	}

	// Finalize encoding
	data, err := s.encoder.End()
	if err != nil {
		return result, fmt.Errorf("end encoding: %w", err)
	}

	// Calculate duration
	durationMs := 0
	if len(input.Frames) > 0 {
		durationMs = input.Frames[len(input.Frames)-1].TimestampMs + input.OutroMs
	}

	result.VideoData = data
	result.DurationMs = durationMs
	result.FileSize = int64(len(data))

	return result, nil
}
