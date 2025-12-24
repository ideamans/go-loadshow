// Package juxtapose provides functionality to combine two videos side by side.
package juxtapose

import (
	"fmt"
	"image"
	"image/draw"

	"github.com/user/loadshow/pkg/adapters/av1decoder"
	"github.com/user/loadshow/pkg/adapters/av1encoder"
	"github.com/user/loadshow/pkg/ports"
)

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

// Combine combines two videos side by side.
// The shorter video will hold its last frame until the longer one finishes.
func Combine(leftPath, rightPath, outputPath string, opts Options) error {
	// Read left video frames
	leftReader := av1decoder.NewMP4Reader()
	leftFrames, err := leftReader.ReadFrames(leftPath)
	leftReader.Close()
	if err != nil {
		return fmt.Errorf("read left video: %w", err)
	}

	if len(leftFrames) == 0 {
		return fmt.Errorf("left video has no frames")
	}

	// Read right video frames
	rightReader := av1decoder.NewMP4Reader()
	rightFrames, err := rightReader.ReadFrames(rightPath)
	rightReader.Close()
	if err != nil {
		return fmt.Errorf("read right video: %w", err)
	}

	if len(rightFrames) == 0 {
		return fmt.Errorf("right video has no frames")
	}

	// Get dimensions
	leftBounds := leftFrames[0].Image.Bounds()
	rightBounds := rightFrames[0].Image.Bounds()

	leftWidth := leftBounds.Dx()
	leftHeight := leftBounds.Dy()
	rightWidth := rightBounds.Dx()
	rightHeight := rightBounds.Dy()

	// Calculate output dimensions
	outputWidth := leftWidth + opts.Gap + rightWidth
	outputHeight := leftHeight
	if rightHeight > outputHeight {
		outputHeight = rightHeight
	}

	// Determine total duration
	leftDuration := leftFrames[len(leftFrames)-1].TimestampMs + leftFrames[len(leftFrames)-1].Duration
	rightDuration := rightFrames[len(rightFrames)-1].TimestampMs + rightFrames[len(rightFrames)-1].Duration
	totalDuration := leftDuration
	if rightDuration > totalDuration {
		totalDuration = rightDuration
	}

	// Create encoder
	encoder := av1encoder.New()
	if err := encoder.Begin(outputWidth, outputHeight, opts.FPS, ports.EncoderOptions{
		Quality: opts.Quality,
		Bitrate: opts.Bitrate,
	}); err != nil {
		return fmt.Errorf("init encoder: %w", err)
	}

	// Generate frames at the output FPS
	frameDurationMs := int(1000.0 / opts.FPS)
	for timestampMs := 0; timestampMs <= totalDuration; timestampMs += frameDurationMs {
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
		rightX := leftWidth + opts.Gap
		rightRect := image.Rect(rightX, rightY, rightX+rightWidth, rightY+rightHeight)
		draw.Draw(output, rightRect, rightFrame.Image, rightFrame.Image.Bounds().Min, draw.Src)

		// Encode frame
		if err := encoder.EncodeFrame(output, timestampMs); err != nil {
			return fmt.Errorf("encode frame at %dms: %w", timestampMs, err)
		}
	}

	// Finalize and write output
	data, err := encoder.End()
	if err != nil {
		return fmt.Errorf("end encoding: %w", err)
	}

	// Write to file
	if err := writeFile(outputPath, data); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

// getFrameAtTime returns the frame at or before the given timestamp.
// If timestamp is past the last frame, returns the last frame.
func getFrameAtTime(frames []av1decoder.VideoFrame, timestampMs int) av1decoder.VideoFrame {
	if len(frames) == 0 {
		return av1decoder.VideoFrame{}
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

func writeFile(path string, data []byte) error {
	f, err := createFile(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}
