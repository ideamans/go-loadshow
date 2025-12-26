package h264encoder

import (
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/user/loadshow/pkg/ports"
)

// createTestImage creates a simple test image with gradient
func createTestImage(width, height int, frameNum int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create a gradient that changes with frame number
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

func TestEncoderBasic(t *testing.T) {
	enc := New()

	width := 320
	height := 240
	fps := 30.0
	opts := ports.EncoderOptions{
		Quality: 25, // Medium quality
	}

	// Initialize encoder
	if err := enc.Begin(width, height, fps, opts); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Encode some frames
	numFrames := 30
	for i := 0; i < numFrames; i++ {
		img := createTestImage(width, height, i)
		timestampMs := i * 1000 / int(fps)

		if err := enc.EncodeFrame(img, timestampMs); err != nil {
			t.Fatalf("EncodeFrame failed at frame %d: %v", i, err)
		}
	}

	// Finalize and get output
	data, err := enc.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("No data produced")
	}

	t.Logf("Encoded %d frames to %d bytes", numFrames, len(data))

	// Verify it starts with ftyp box (MP4 signature)
	if len(data) < 8 {
		t.Fatal("Output too small")
	}

	// Check for 'ftyp' box
	if string(data[4:8]) != "ftyp" {
		t.Errorf("Expected ftyp box, got: %s", string(data[4:8]))
	}
}

func TestEncoderHighQuality(t *testing.T) {
	enc := New()

	width := 640
	height := 480
	fps := 30.0
	opts := ports.EncoderOptions{
		Quality: 10, // High quality
		Bitrate: 2000,
	}

	if err := enc.Begin(width, height, fps, opts); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Encode a few frames
	for i := 0; i < 10; i++ {
		img := createTestImage(width, height, i)
		if err := enc.EncodeFrame(img, i*33); err != nil {
			t.Fatalf("EncodeFrame failed: %v", err)
		}
	}

	data, err := enc.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	t.Logf("High quality output: %d bytes", len(data))
}

func TestEncoderLowQuality(t *testing.T) {
	enc := New()

	width := 320
	height := 240
	fps := 15.0
	opts := ports.EncoderOptions{
		Quality: 50, // Low quality
	}

	if err := enc.Begin(width, height, fps, opts); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	for i := 0; i < 15; i++ {
		img := createTestImage(width, height, i)
		if err := enc.EncodeFrame(img, i*67); err != nil {
			t.Fatalf("EncodeFrame failed: %v", err)
		}
	}

	data, err := enc.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	t.Logf("Low quality output: %d bytes", len(data))
}

func TestEncoderSingleFrame(t *testing.T) {
	enc := New()

	if err := enc.Begin(100, 100, 1.0, ports.EncoderOptions{}); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	img := createTestImage(100, 100, 0)
	if err := enc.EncodeFrame(img, 0); err != nil {
		t.Fatalf("EncodeFrame failed: %v", err)
	}

	data, err := enc.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("No data produced for single frame")
	}

	t.Logf("Single frame output: %d bytes", len(data))
}

func TestEncoderWriteToFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file write test in short mode")
	}

	enc := New()

	width := 320
	height := 240
	fps := 30.0

	if err := enc.Begin(width, height, fps, ports.EncoderOptions{Quality: 25}); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Encode 60 frames (2 seconds)
	for i := 0; i < 60; i++ {
		img := createTestImage(width, height, i)
		if err := enc.EncodeFrame(img, i*33); err != nil {
			t.Fatalf("EncodeFrame failed: %v", err)
		}
	}

	data, err := enc.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	// Write to temp file for manual inspection
	tmpFile, err := os.CreateTemp("", "h264test_*.mp4")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}
	tmpFile.Close()

	t.Logf("Wrote test video to: %s (%d bytes)", tmpFile.Name(), len(data))

	// Verify file exists and has content
	info, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if info.Size() == 0 {
		t.Fatal("Output file is empty")
	}
}

func TestEncoderNotInitialized(t *testing.T) {
	enc := New()

	// Try to encode without initialization
	img := createTestImage(100, 100, 0)
	err := enc.EncodeFrame(img, 0)
	if err != ErrNotInitialized {
		t.Errorf("Expected ErrNotInitialized, got: %v", err)
	}

	// Try to end without initialization
	_, err = enc.End()
	if err != ErrNotInitialized {
		t.Errorf("Expected ErrNotInitialized, got: %v", err)
	}
}

func BenchmarkEncode320x240(b *testing.B) {
	enc := New()
	width, height := 320, 240

	if err := enc.Begin(width, height, 30.0, ports.EncoderOptions{Quality: 25}); err != nil {
		b.Fatalf("Begin failed: %v", err)
	}

	img := createTestImage(width, height, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := enc.EncodeFrame(img, i*33); err != nil {
			b.Fatalf("EncodeFrame failed: %v", err)
		}
	}
	b.StopTimer()

	enc.End()
}

func BenchmarkEncode640x480(b *testing.B) {
	enc := New()
	width, height := 640, 480

	if err := enc.Begin(width, height, 30.0, ports.EncoderOptions{Quality: 25}); err != nil {
		b.Fatalf("Begin failed: %v", err)
	}

	img := createTestImage(width, height, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := enc.EncodeFrame(img, i*33); err != nil {
			b.Fatalf("EncodeFrame failed: %v", err)
		}
	}
	b.StopTimer()

	enc.End()
}

func BenchmarkEncode1280x720(b *testing.B) {
	enc := New()
	width, height := 1280, 720

	if err := enc.Begin(width, height, 30.0, ports.EncoderOptions{Quality: 25}); err != nil {
		b.Fatalf("Begin failed: %v", err)
	}

	img := createTestImage(width, height, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := enc.EncodeFrame(img, i*33); err != nil {
			b.Fatalf("EncodeFrame failed: %v", err)
		}
	}
	b.StopTimer()

	enc.End()
}
