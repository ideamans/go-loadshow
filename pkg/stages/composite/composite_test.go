package composite

import (
	"context"
	"image"
	"testing"

	"github.com/user/loadshow/pkg/mocks"
	"github.com/user/loadshow/pkg/pipeline"
	"github.com/user/loadshow/pkg/ports"
	"github.com/user/loadshow/pkg/stages/layout"
)

func TestStage_Execute(t *testing.T) {
	mockRenderer := &mocks.Renderer{
		DecodeImageFunc: func(data []byte, format ports.ImageFormat) (image.Image, error) {
			// Return a test image
			return image.NewRGBA(image.Rect(0, 0, 142, 1740)), nil
		},
	}
	mockSink := mocks.NewDebugSink(false)

	stage := NewStage(mockRenderer, mockSink, 2)

	// Create layout
	layoutInput := pipeline.DefaultLayoutInput()
	layoutResult := layout.ComputeLayout(layoutInput)

	// Create raw frames
	rawFrames := []pipeline.RawFrame{
		{TimestampMs: 0, ImageData: []byte{0xFF, 0xD8}},
		{TimestampMs: 100, ImageData: []byte{0xFF, 0xD8}},
		{TimestampMs: 200, ImageData: []byte{0xFF, 0xD8}},
	}

	input := pipeline.CompositeInput{
		RawFrames:    rawFrames,
		Layout:       layoutResult,
		Banner:       nil,
		Theme:        pipeline.DefaultCompositeTheme(),
		ShowProgress: true,
		TotalTimeMs:  200,
	}

	result, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check number of frames
	if len(result.Frames) != 3 {
		t.Errorf("expected 3 frames, got %d", len(result.Frames))
	}

	// Check frames are in order
	for i, frame := range result.Frames {
		expectedTs := i * 100
		if frame.TimestampMs != expectedTs {
			t.Errorf("frame %d: expected timestamp %d, got %d", i, expectedTs, frame.TimestampMs)
		}
		if frame.Image == nil {
			t.Errorf("frame %d: image is nil", i)
		}
	}
}

func TestStage_Execute_EmptyFrames(t *testing.T) {
	mockRenderer := &mocks.Renderer{}
	mockSink := mocks.NewDebugSink(false)

	stage := NewStage(mockRenderer, mockSink, 2)

	input := pipeline.CompositeInput{
		RawFrames: []pipeline.RawFrame{},
	}

	result, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Frames) != 0 {
		t.Errorf("expected 0 frames, got %d", len(result.Frames))
	}
}

func TestStage_Execute_WithBanner(t *testing.T) {
	mockRenderer := &mocks.Renderer{
		DecodeImageFunc: func(data []byte, format ports.ImageFormat) (image.Image, error) {
			return image.NewRGBA(image.Rect(0, 0, 142, 1740)), nil
		},
	}
	mockSink := mocks.NewDebugSink(false)

	stage := NewStage(mockRenderer, mockSink, 2)

	// Create layout with banner
	layoutInput := pipeline.DefaultLayoutInput()
	layoutInput.BannerHeight = 80
	layoutResult := layout.ComputeLayout(layoutInput)

	// Create banner
	bannerImg := image.NewRGBA(image.Rect(0, 0, 472, 80))
	banner := &pipeline.BannerResult{Image: bannerImg}

	input := pipeline.CompositeInput{
		RawFrames: []pipeline.RawFrame{
			{TimestampMs: 0, ImageData: []byte{0xFF, 0xD8}},
		},
		Layout:       layoutResult,
		Banner:       banner,
		Theme:        pipeline.DefaultCompositeTheme(),
		ShowProgress: false,
	}

	result, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Frames) != 1 {
		t.Errorf("expected 1 frame, got %d", len(result.Frames))
	}
}

func TestStage_Execute_WithDebugSink(t *testing.T) {
	mockRenderer := &mocks.Renderer{
		DecodeImageFunc: func(data []byte, format ports.ImageFormat) (image.Image, error) {
			return image.NewRGBA(image.Rect(0, 0, 142, 1740)), nil
		},
	}
	mockSink := mocks.NewDebugSink(true)

	stage := NewStage(mockRenderer, mockSink, 2)

	layoutInput := pipeline.DefaultLayoutInput()
	layoutResult := layout.ComputeLayout(layoutInput)

	input := pipeline.CompositeInput{
		RawFrames: []pipeline.RawFrame{
			{TimestampMs: 0, ImageData: []byte{0xFF, 0xD8}},
			{TimestampMs: 100, ImageData: []byte{0xFF, 0xD8}},
		},
		Layout: layoutResult,
		Theme:  pipeline.DefaultCompositeTheme(),
	}

	_, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that frames were saved to sink
	if len(mockSink.ComposedFrames) != 2 {
		t.Errorf("expected 2 composed frames in sink, got %d", len(mockSink.ComposedFrames))
	}
}

func TestExtractSubImage(t *testing.T) {
	// Create a test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Extract a sub-region
	sub := extractSubImage(img, 10, 10, 50, 50)
	if sub == nil {
		t.Fatal("extractSubImage returned nil")
	}

	bounds := sub.Bounds()
	if bounds.Dx() != 50 || bounds.Dy() != 50 {
		t.Errorf("expected 50x50, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestExtractSubImage_OutOfBounds(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Request extends beyond image bounds
	sub := extractSubImage(img, 80, 80, 50, 50)
	if sub == nil {
		t.Fatal("extractSubImage returned nil")
	}

	// Should be clamped to 20x20
	bounds := sub.Bounds()
	if bounds.Dx() != 20 || bounds.Dy() != 20 {
		t.Errorf("expected 20x20, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestExtractSubImage_NegativeResult(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Request entirely outside image
	sub := extractSubImage(img, 200, 200, 50, 50)
	if sub != nil {
		t.Error("expected nil for out-of-bounds request")
	}
}
