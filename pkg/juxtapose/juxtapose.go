// Package juxtapose provides functionality to combine two videos side by side.
package juxtapose

import (
	"context"
	"fmt"
	"image"
	"image/draw"

	"github.com/user/loadshow/pkg/ports"
)

// Input contains the input parameters for juxtapose operation.
type Input struct {
	// LeftPath is the file path of the left video.
	LeftPath string
	// RightPath is the file path of the right video.
	RightPath string
	// OutputPath is the file path for the output video.
	OutputPath string
}

// Result contains the result of juxtapose operation.
type Result struct {
	// OutputPath is the path where the output was written.
	OutputPath string
	// FrameCount is the number of frames in the output video.
	FrameCount int
	// DurationMs is the duration of the output video in milliseconds.
	DurationMs int
}

// Options configures the juxtapose operation.
type Options struct {
	// Gap is the horizontal gap between the two videos in pixels.
	Gap int
	// FPS is the output frame rate.
	FPS float64
	// Quality is the encoding quality (CRF 0-63, lower is better).
	Quality int
	// Bitrate is the target bitrate in kbps (0 = auto).
	Bitrate int
}

// DefaultOptions returns default options.
func DefaultOptions() Options {
	return Options{
		Gap:     10,
		FPS:     30.0,
		Quality: 30,
		Bitrate: 0,
	}
}

// Stage implements the juxtapose operation with dependency injection.
type Stage struct {
	decoder ports.VideoDecoder
	encoder ports.VideoEncoder
	fs      ports.FileSystem
	logger  ports.Logger
	opts    Options
}

// New creates a new juxtapose stage with the given dependencies.
func New(
	decoder ports.VideoDecoder,
	encoder ports.VideoEncoder,
	fs ports.FileSystem,
	logger ports.Logger,
	opts Options,
) *Stage {
	return &Stage{
		decoder: decoder,
		encoder: encoder,
		fs:      fs,
		logger:  logger.WithComponent("juxtapose"),
		opts:    opts,
	}
}

// Execute combines two videos side by side.
// The shorter video will hold its last frame until the longer one finishes.
func (s *Stage) Execute(ctx context.Context, input Input) (Result, error) {
	result := Result{
		OutputPath: input.OutputPath,
	}

	s.logger.Debug("Reading left video: %s", input.LeftPath)

	// Read left video frames
	leftFrames, err := s.decoder.ReadFrames(input.LeftPath)
	if err != nil {
		return result, fmt.Errorf("read left video: %w", err)
	}

	if len(leftFrames) == 0 {
		return result, fmt.Errorf("left video has no frames")
	}

	s.logger.Debug("Left video: %d frames", len(leftFrames))

	s.logger.Debug("Reading right video: %s", input.RightPath)

	// Read right video frames
	rightFrames, err := s.decoder.ReadFrames(input.RightPath)
	if err != nil {
		return result, fmt.Errorf("read right video: %w", err)
	}

	if len(rightFrames) == 0 {
		return result, fmt.Errorf("right video has no frames")
	}

	s.logger.Debug("Right video: %d frames", len(rightFrames))

	// Get dimensions
	leftBounds := leftFrames[0].Image.Bounds()
	rightBounds := rightFrames[0].Image.Bounds()

	leftWidth := leftBounds.Dx()
	leftHeight := leftBounds.Dy()
	rightWidth := rightBounds.Dx()
	rightHeight := rightBounds.Dy()

	// Calculate output dimensions
	outputWidth := leftWidth + s.opts.Gap + rightWidth
	outputHeight := leftHeight
	if rightHeight > outputHeight {
		outputHeight = rightHeight
	}

	s.logger.Debug("Output dimensions: %dx%d", outputWidth, outputHeight)

	// Determine total duration
	leftDuration := leftFrames[len(leftFrames)-1].TimestampMs + leftFrames[len(leftFrames)-1].Duration
	rightDuration := rightFrames[len(rightFrames)-1].TimestampMs + rightFrames[len(rightFrames)-1].Duration
	totalDuration := leftDuration
	if rightDuration > totalDuration {
		totalDuration = rightDuration
	}

	result.DurationMs = totalDuration

	// Initialize encoder
	if err := s.encoder.Begin(outputWidth, outputHeight, s.opts.FPS, ports.EncoderOptions{
		Quality: s.opts.Quality,
		Bitrate: s.opts.Bitrate,
	}); err != nil {
		return result, fmt.Errorf("init encoder: %w", err)
	}

	// Generate frames at the output FPS
	frameDurationMs := int(1000.0 / s.opts.FPS)
	frameCount := 0

	for timestampMs := 0; timestampMs <= totalDuration; timestampMs += frameDurationMs {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Get left frame at this timestamp
		leftFrame := getFrameAtTime(leftFrames, timestampMs)
		rightFrame := getFrameAtTime(rightFrames, timestampMs)

		// Composite frames side by side
		output := image.NewRGBA(image.Rect(0, 0, outputWidth, outputHeight))

		// Fill with black background
		draw.Draw(output, output.Bounds(), image.Black, image.Point{}, draw.Src)

		// Draw left video (vertically centered)
		leftY := (outputHeight - leftHeight) / 2
		leftRect := image.Rect(0, leftY, leftWidth, leftY+leftHeight)
		draw.Draw(output, leftRect, leftFrame.Image, leftFrame.Image.Bounds().Min, draw.Src)

		// Draw right video (vertically centered)
		rightY := (outputHeight - rightHeight) / 2
		rightX := leftWidth + s.opts.Gap
		rightRect := image.Rect(rightX, rightY, rightX+rightWidth, rightY+rightHeight)
		draw.Draw(output, rightRect, rightFrame.Image, rightFrame.Image.Bounds().Min, draw.Src)

		// Encode frame
		if err := s.encoder.EncodeFrame(output, timestampMs); err != nil {
			return result, fmt.Errorf("encode frame at %dms: %w", timestampMs, err)
		}

		frameCount++
	}

	result.FrameCount = frameCount

	s.logger.Debug("Encoded %d frames", frameCount)

	// Finalize and get video data
	data, err := s.encoder.End()
	if err != nil {
		return result, fmt.Errorf("end encoding: %w", err)
	}

	s.logger.Debug("Writing output: %s (%d bytes)", input.OutputPath, len(data))

	// Write to file using FileSystem
	if err := s.fs.WriteFile(input.OutputPath, data); err != nil {
		return result, fmt.Errorf("write output: %w", err)
	}

	s.logger.Info("Juxtapose completed: %s", input.OutputPath)

	return result, nil
}

// getFrameAtTime returns the frame at or before the given timestamp.
// If timestamp is past the last frame, returns the last frame.
func getFrameAtTime(frames []ports.VideoFrame, timestampMs int) ports.VideoFrame {
	if len(frames) == 0 {
		return ports.VideoFrame{}
	}

	// Find the frame that contains this timestamp
	for i := len(frames) - 1; i >= 0; i-- {
		if frames[i].TimestampMs <= timestampMs {
			return frames[i]
		}
	}

	// Return first frame if timestamp is before all frames
	return frames[0]
}
