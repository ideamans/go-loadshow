package ggrenderer

import (
	"image"
	"image/color"
	"testing"

	"github.com/user/loadshow/pkg/ports"
)

func TestRenderer_CreateCanvas(t *testing.T) {
	r := New()

	canvas := r.CreateCanvas(100, 100, color.White)
	if canvas == nil {
		t.Fatal("expected canvas to be created")
	}

	img := canvas.ToImage()
	bounds := img.Bounds()

	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("expected 100x100, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestRenderer_EncodeDecodeJPEG(t *testing.T) {
	r := New()

	// Create test image
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	// Encode
	data, err := r.EncodeImage(img, ports.FormatJPEG, 80)
	if err != nil {
		t.Fatalf("EncodeImage failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data")
	}

	// Decode
	decoded, err := r.DecodeImage(data, ports.FormatJPEG)
	if err != nil {
		t.Fatalf("DecodeImage failed: %v", err)
	}

	bounds := decoded.Bounds()
	if bounds.Dx() != 50 || bounds.Dy() != 50 {
		t.Errorf("expected 50x50, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestRenderer_EncodeDecodePNG(t *testing.T) {
	r := New()

	img := image.NewRGBA(image.Rect(0, 0, 30, 30))

	// Encode
	data, err := r.EncodeImage(img, ports.FormatPNG, 0)
	if err != nil {
		t.Fatalf("EncodeImage failed: %v", err)
	}

	// Decode
	decoded, err := r.DecodeImage(data, ports.FormatPNG)
	if err != nil {
		t.Fatalf("DecodeImage failed: %v", err)
	}

	bounds := decoded.Bounds()
	if bounds.Dx() != 30 || bounds.Dy() != 30 {
		t.Errorf("expected 30x30, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestRenderer_ResizeImage(t *testing.T) {
	r := New()

	// Create 100x100 image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Resize to 50x50
	resized := r.ResizeImage(img, 50, 50)

	bounds := resized.Bounds()
	if bounds.Dx() != 50 || bounds.Dy() != 50 {
		t.Errorf("expected 50x50, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestCanvas_DrawRect(t *testing.T) {
	r := New()
	canvas := r.CreateCanvas(100, 100, color.White)

	// Draw red rectangle
	canvas.DrawRect(10, 10, 30, 30, color.RGBA{R: 255, A: 255})

	img := canvas.ToImage()

	// Check that pixel inside rectangle is red
	c := img.At(20, 20)
	red, _, _, _ := c.RGBA()
	if red == 0 {
		t.Error("expected red pixel inside rectangle")
	}
}

func TestCanvas_DrawRectStroke(t *testing.T) {
	r := New()
	canvas := r.CreateCanvas(100, 100, color.White)

	// Draw rectangle stroke
	canvas.DrawRectStroke(10, 10, 30, 30, color.Black, 2)

	img := canvas.ToImage()

	// Check border pixel
	c := img.At(10, 10)
	_, _, _, a := c.RGBA()
	if a == 0 {
		t.Error("expected non-transparent pixel on border")
	}
}

func TestCanvas_DrawImage(t *testing.T) {
	r := New()
	canvas := r.CreateCanvas(100, 100, color.White)

	// Create small red image
	small := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			small.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}

	// Draw at position (10, 10)
	canvas.DrawImage(small, 10, 10)

	img := canvas.ToImage()

	// Check pixel at (15, 15) should be red
	c := img.At(15, 15)
	red, _, _, _ := c.RGBA()
	if red == 0 {
		t.Error("expected red pixel from drawn image")
	}
}

func TestCanvas_DrawLine(t *testing.T) {
	r := New()
	canvas := r.CreateCanvas(100, 100, color.White)

	// Draw black line
	canvas.DrawLine(0, 50, 100, 50, color.Black, 2)

	img := canvas.ToImage()

	// Check pixel on line
	c := img.At(50, 50)
	r1, g1, b1, _ := c.RGBA()
	// Should be dark (not white)
	if r1 == 65535 && g1 == 65535 && b1 == 65535 {
		t.Error("expected non-white pixel on line")
	}
}

func TestCanvas_DrawText(t *testing.T) {
	r := New()
	canvas := r.CreateCanvas(200, 50, color.White)

	style := ports.TextStyle{
		FontSize: 14,
		Color:    color.Black,
		Align:    ports.AlignLeft,
	}

	// Should not panic
	canvas.DrawText("Hello World", 10, 25, style)

	img := canvas.ToImage()
	if img == nil {
		t.Error("expected image to be created")
	}
}
