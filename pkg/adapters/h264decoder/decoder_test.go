package h264decoder

import (
	"bytes"
	"image"
	"image/color"
	"runtime"
	"testing"

	"github.com/user/loadshow/pkg/adapters/h264encoder"
	"github.com/user/loadshow/pkg/ports"
)

// createTestImage creates a simple test image with gradient
func createTestImage(width, height int, frameNum int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255 / width + frameNum*10) % 256)
			g := uint8((y * 255 / height + frameNum*5) % 256)
			b := uint8((x + y + frameNum*3) % 256)
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	return img
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Media Foundation H.264 decoder not available on Windows Server CI")
	}

	// First, encode some frames
	enc := h264encoder.New()

	width := 256
	height := 192
	fps := 30.0
	numFrames := 5

	if err := enc.Begin(width, height, fps, ports.EncoderOptions{Quality: 20}); err != nil {
		t.Fatalf("encoder Begin failed: %v", err)
	}

	for i := 0; i < numFrames; i++ {
		img := createTestImage(width, height, i)
		timestampMs := i * 1000 / int(fps)

		if err := enc.EncodeFrame(img, timestampMs); err != nil {
			t.Fatalf("EncodeFrame failed at frame %d: %v", i, err)
		}
	}

	mp4Data, err := enc.End()
	if err != nil {
		t.Fatalf("encoder End failed: %v", err)
	}

	t.Logf("Encoded %d frames to %d bytes", numFrames, len(mp4Data))

	// Now decode the frames
	reader := NewMP4Reader()
	defer reader.Close()

	frames, err := reader.ReadFramesFromReader(bytes.NewReader(mp4Data))
	if err != nil {
		t.Fatalf("ReadFramesFromReader failed: %v", err)
	}

	t.Logf("Decoded %d frames", len(frames))

	if len(frames) == 0 {
		t.Fatal("No frames decoded")
	}

	// Verify first frame dimensions
	firstFrame := frames[0]
	bounds := firstFrame.Image.Bounds()
	if bounds.Dx() != width || bounds.Dy() != height {
		t.Errorf("Expected dimensions %dx%d, got %dx%d", width, height, bounds.Dx(), bounds.Dy())
	}

	// Check timestamps are increasing
	lastTs := -1
	for i, frame := range frames {
		if frame.TimestampMs <= lastTs {
			t.Errorf("Frame %d has non-increasing timestamp: %d <= %d", i, frame.TimestampMs, lastTs)
		}
		lastTs = frame.TimestampMs
	}
}

func TestExtractFrames(t *testing.T) {
	// Encode some frames
	enc := h264encoder.New()

	if err := enc.Begin(256, 192, 15.0, ports.EncoderOptions{}); err != nil {
		t.Fatalf("encoder Begin failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		img := createTestImage(256, 192, i)
		if err := enc.EncodeFrame(img, i*67); err != nil {
			t.Fatalf("EncodeFrame failed: %v", err)
		}
	}

	mp4Data, err := enc.End()
	if err != nil {
		t.Fatalf("encoder End failed: %v", err)
	}

	// Extract raw frames
	rawFrames, err := ExtractFrames(mp4Data)
	if err != nil {
		t.Fatalf("ExtractFrames failed: %v", err)
	}

	t.Logf("Extracted %d raw frames", len(rawFrames))

	if len(rawFrames) == 0 {
		t.Fatal("No frames extracted")
	}

	// First frame should be a keyframe
	if !rawFrames[0].IsKeyframe {
		t.Error("First frame should be a keyframe")
	}

	// All frames should have data
	for i, frame := range rawFrames {
		if len(frame.Data) == 0 {
			t.Errorf("Frame %d has no data", i)
		}
	}
}
