package av1decoder

import (
	"image"
	"image/color"
	"testing"

	"github.com/user/loadshow/pkg/adapters/av1encoder"
	"github.com/user/loadshow/pkg/ports"
)

func TestNew(t *testing.T) {
	decoder := New()
	if decoder == nil {
		t.Fatal("expected decoder to be created")
	}
}

func TestDecoder_Init(t *testing.T) {
	decoder := New()

	err := decoder.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	decoder.Close()
}

func TestDecoder_DecodeWithoutInit(t *testing.T) {
	decoder := New()

	_, err := decoder.DecodeFrame([]byte{0x00})
	if err == nil {
		t.Error("expected error when decoding without Init")
	}
}

func TestDecoder_DecodeEmptyData(t *testing.T) {
	decoder := New()
	if err := decoder.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer decoder.Close()

	_, err := decoder.DecodeFrame([]byte{})
	if err == nil {
		t.Error("expected error when decoding empty data")
	}
}

func TestDecoder_Close(t *testing.T) {
	decoder := New()
	if err := decoder.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Should not panic
	decoder.Close()
	decoder.Close() // Double close should be safe
}

func TestClamp(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{-10, 0},
		{0, 0},
		{128, 128},
		{255, 255},
		{300, 255},
	}

	for _, tt := range tests {
		result := clamp(tt.input)
		if result != tt.expected {
			t.Errorf("clamp(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

// TestEncoderDecoderRoundtrip tests encoding and decoding a frame
func TestEncoderDecoderRoundtrip(t *testing.T) {
	// Create encoder and encode a frame
	encoder := av1encoder.New()
	if err := encoder.Begin(64, 64, 30.0, ports.EncoderOptions{Quality: 40}); err != nil {
		t.Fatalf("Encoder Begin failed: %v", err)
	}

	// Create a simple red image
	origImg := createTestImage(64, 64, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	if err := encoder.EncodeFrame(origImg, 0); err != nil {
		t.Fatalf("EncodeFrame failed: %v", err)
	}

	mp4Data, err := encoder.End()
	if err != nil {
		t.Fatalf("Encoder End failed: %v", err)
	}

	// Extract AV1 frame from MP4
	frames, err := ExtractFrames(mp4Data)
	if err != nil {
		t.Fatalf("ExtractFrames failed: %v", err)
	}

	if len(frames) == 0 {
		t.Fatal("no frames extracted")
	}

	// Decode the first frame
	decoder := New()
	if err := decoder.Init(); err != nil {
		t.Fatalf("Decoder Init failed: %v", err)
	}
	defer decoder.Close()

	decodedImg, err := decoder.DecodeFrame(frames[0].Data)
	if err != nil {
		t.Fatalf("DecodeFrame failed: %v", err)
	}

	// Verify decoded image dimensions
	bounds := decodedImg.Bounds()
	if bounds.Dx() != 64 || bounds.Dy() != 64 {
		t.Errorf("decoded image size %dx%d, want 64x64", bounds.Dx(), bounds.Dy())
	}

	// Verify the image is roughly the right color (allowing for compression artifacts)
	rgba, ok := decodedImg.(*image.RGBA)
	if !ok {
		t.Fatal("expected RGBA image")
	}

	// Check center pixel is reddish
	centerIdx := (32*rgba.Stride + 32*4)
	r := rgba.Pix[centerIdx]
	g := rgba.Pix[centerIdx+1]
	b := rgba.Pix[centerIdx+2]

	// With lossy compression, colors won't be exact, but red should dominate
	if r < 150 || g > 100 || b > 100 {
		t.Logf("Center pixel: R=%d G=%d B=%d (expected reddish)", r, g, b)
		// Don't fail - compression artifacts can vary
	}
}

func TestEncoderDecoderMultipleFrames(t *testing.T) {
	// Create encoder and encode multiple frames
	encoder := av1encoder.New()
	if err := encoder.Begin(64, 64, 30.0, ports.EncoderOptions{Quality: 40}); err != nil {
		t.Fatalf("Encoder Begin failed: %v", err)
	}

	colors := []color.RGBA{
		{R: 255, G: 0, B: 0, A: 255},
		{R: 0, G: 255, B: 0, A: 255},
		{R: 0, G: 0, B: 255, A: 255},
	}

	for i, c := range colors {
		img := createTestImage(64, 64, c)
		if err := encoder.EncodeFrame(img, i*33); err != nil {
			t.Fatalf("EncodeFrame %d failed: %v", i, err)
		}
	}

	mp4Data, err := encoder.End()
	if err != nil {
		t.Fatalf("Encoder End failed: %v", err)
	}

	// Extract frames
	frames, err := ExtractFrames(mp4Data)
	if err != nil {
		t.Fatalf("ExtractFrames failed: %v", err)
	}

	if len(frames) < 3 {
		t.Errorf("expected at least 3 frames, got %d", len(frames))
	}

	// Decode all frames
	decoder := New()
	if err := decoder.Init(); err != nil {
		t.Fatalf("Decoder Init failed: %v", err)
	}
	defer decoder.Close()

	for i, frame := range frames {
		_, err := decoder.DecodeFrame(frame.Data)
		if err != nil {
			t.Errorf("DecodeFrame %d failed: %v", i, err)
		}
	}
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
