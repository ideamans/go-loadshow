package banner

import (
	"context"
	"testing"

	"github.com/user/loadshow/pkg/mocks"
	"github.com/user/loadshow/pkg/pipeline"
)

func TestStage_Execute(t *testing.T) {
	mockRenderer := &mocks.Renderer{}
	mockSink := mocks.NewDebugSink(false)

	stage := NewStage(mockRenderer, mockSink)

	input := pipeline.BannerInput{
		Width:      400,
		Height:     80,
		URL:        "https://example.com/page",
		Title:      "Example Page Title",
		LoadTimeMs: 2500,
		TotalBytes: 1024 * 500, // 500 KB
		Theme:      pipeline.DefaultBannerTheme(),
	}

	result, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that image was created
	if result.Image == nil {
		t.Error("expected image to be created")
	}

	// Check image dimensions
	bounds := result.Image.Bounds()
	if bounds.Dx() != input.Width || bounds.Dy() != input.Height {
		t.Errorf("expected image size %dx%d, got %dx%d",
			input.Width, input.Height, bounds.Dx(), bounds.Dy())
	}
}

func TestStage_Execute_WithDebugSink(t *testing.T) {
	mockRenderer := &mocks.Renderer{}
	mockSink := mocks.NewDebugSink(true)

	stage := NewStage(mockRenderer, mockSink)

	input := pipeline.BannerInput{
		Width:      400,
		Height:     80,
		URL:        "https://example.com",
		Title:      "Test",
		LoadTimeMs: 1000,
		TotalBytes: 1024,
		Theme:      pipeline.DefaultBannerTheme(),
	}

	_, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that banner was saved to sink
	if mockSink.Banner == nil {
		t.Error("expected banner to be saved to debug sink")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is a long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q",
				tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1024 * 1024 * 1024 * 2, "2.00 GB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %q, want %q",
				tt.bytes, result, tt.expected)
		}
	}
}
