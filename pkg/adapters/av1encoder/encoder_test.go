package av1encoder

import (
	"image"
	"image/color"
	"testing"

	"github.com/user/loadshow/pkg/ports"
)

func TestNew(t *testing.T) {
	encoder := New()
	if encoder == nil {
		t.Fatal("expected encoder to be created")
	}
}

func TestEncoder_Begin(t *testing.T) {
	encoder := New()

	err := encoder.Begin(128, 128, 30.0, ports.EncoderOptions{
		Quality: 30,
		Bitrate: 1000,
	})
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Cleanup
	encoder.cleanup()
}

func TestEncoder_EncodeFrame(t *testing.T) {
	encoder := New()

	err := encoder.Begin(64, 64, 30.0, ports.EncoderOptions{
		Quality: 40,
	})
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Create a simple test image
	img := createTestImage(64, 64, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	err = encoder.EncodeFrame(img, 0)
	if err != nil {
		t.Fatalf("EncodeFrame failed: %v", err)
	}

	if encoder.frameCount != 1 {
		t.Errorf("expected frameCount 1, got %d", encoder.frameCount)
	}

	encoder.cleanup()
}

func TestEncoder_EncodeMultipleFrames(t *testing.T) {
	encoder := New()

	err := encoder.Begin(64, 64, 30.0, ports.EncoderOptions{
		Quality: 40,
	})
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	colors := []color.RGBA{
		{R: 255, G: 0, B: 0, A: 255},   // Red
		{R: 0, G: 255, B: 0, A: 255},   // Green
		{R: 0, G: 0, B: 255, A: 255},   // Blue
		{R: 255, G: 255, B: 0, A: 255}, // Yellow
		{R: 255, G: 0, B: 255, A: 255}, // Magenta
	}

	for i, c := range colors {
		img := createTestImage(64, 64, c)
		err = encoder.EncodeFrame(img, i*33) // ~30fps
		if err != nil {
			t.Fatalf("EncodeFrame %d failed: %v", i, err)
		}
	}

	if encoder.frameCount != 5 {
		t.Errorf("expected frameCount 5, got %d", encoder.frameCount)
	}

	encoder.cleanup()
}

func TestEncoder_End(t *testing.T) {
	encoder := New()

	err := encoder.Begin(64, 64, 30.0, ports.EncoderOptions{
		Quality: 40,
	})
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Encode a few frames
	for i := 0; i < 5; i++ {
		img := createTestImage(64, 64, color.RGBA{
			R: uint8(i * 50),
			G: uint8(255 - i*50),
			B: 128,
			A: 255,
		})
		if err := encoder.EncodeFrame(img, i*33); err != nil {
			t.Fatalf("EncodeFrame %d failed: %v", i, err)
		}
	}

	// End encoding and get MP4 data
	mp4Data, err := encoder.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	if len(mp4Data) == 0 {
		t.Error("expected non-empty MP4 data")
	}

	// Check MP4 signature (ftyp box)
	if len(mp4Data) < 8 {
		t.Fatal("MP4 data too short")
	}

	// MP4 files start with ftyp box
	ftypSignature := string(mp4Data[4:8])
	if ftypSignature != "ftyp" {
		t.Errorf("expected ftyp signature, got %q", ftypSignature)
	}
}

func TestEncoder_EndWithoutFrames(t *testing.T) {
	encoder := New()

	err := encoder.Begin(64, 64, 30.0, ports.EncoderOptions{})
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Try to end without encoding any frames
	_, err = encoder.End()
	if err == nil {
		t.Error("expected error when ending without frames")
	}
}

func TestEncoder_EncodeWithoutBegin(t *testing.T) {
	encoder := New()

	img := createTestImage(64, 64, color.RGBA{R: 255, A: 255})
	err := encoder.EncodeFrame(img, 0)
	if err == nil {
		t.Error("expected error when encoding without Begin")
	}
}

func TestEncoder_DifferentResolutions(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"small", 64, 64},
		{"medium", 256, 256},
		{"wide", 320, 180},
		{"tall", 180, 320},
		{"standard", 512, 640},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := New()

			err := encoder.Begin(tt.width, tt.height, 30.0, ports.EncoderOptions{
				Quality: 45,
			})
			if err != nil {
				t.Fatalf("Begin failed: %v", err)
			}

			img := createTestImage(tt.width, tt.height, color.RGBA{R: 100, G: 150, B: 200, A: 255})
			if err := encoder.EncodeFrame(img, 0); err != nil {
				t.Fatalf("EncodeFrame failed: %v", err)
			}

			mp4Data, err := encoder.End()
			if err != nil {
				t.Fatalf("End failed: %v", err)
			}

			if len(mp4Data) == 0 {
				t.Error("expected non-empty MP4 data")
			}
		})
	}
}

func TestEncoder_QualitySettings(t *testing.T) {
	tests := []struct {
		name    string
		quality int
	}{
		{"high quality", 20},
		{"medium quality", 35},
		{"low quality", 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := New()

			err := encoder.Begin(64, 64, 30.0, ports.EncoderOptions{
				Quality: tt.quality,
			})
			if err != nil {
				t.Fatalf("Begin failed: %v", err)
			}

			img := createTestImage(64, 64, color.RGBA{R: 128, G: 128, B: 128, A: 255})
			if err := encoder.EncodeFrame(img, 0); err != nil {
				t.Fatalf("EncodeFrame failed: %v", err)
			}

			mp4Data, err := encoder.End()
			if err != nil {
				t.Fatalf("End failed: %v", err)
			}

			if len(mp4Data) == 0 {
				t.Error("expected non-empty MP4 data")
			}
		})
	}
}

func TestEncoder_ImplementsInterface(t *testing.T) {
	var _ ports.VideoEncoder = (*Encoder)(nil)
}

// createTestImage creates a solid color test image
func createTestImage(width, height int, c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// Benchmark tests
func BenchmarkEncoder_EncodeFrame(b *testing.B) {
	encoder := New()
	if err := encoder.Begin(256, 256, 30.0, ports.EncoderOptions{Quality: 40}); err != nil {
		b.Fatalf("Begin failed: %v", err)
	}
	defer encoder.cleanup()

	img := createTestImage(256, 256, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := encoder.EncodeFrame(img, i*33); err != nil {
			b.Fatalf("EncodeFrame failed: %v", err)
		}
	}
}
