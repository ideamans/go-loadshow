package banner

import (
	"context"
	"testing"

	"github.com/user/loadshow/pkg/mocks"
	"github.com/user/loadshow/pkg/pipeline"
)

func TestStage_Execute(t *testing.T) {
	mockCapturer := mocks.NewHTMLCapturer()
	mockSink := mocks.NewDebugSink(false)

	stage := NewStage(mockCapturer, mockSink)

	input := pipeline.BannerInput{
		Width:      400,
		Height:     80,
		URL:        "https://example.com/page",
		Title:      "Example Page Title",
		LoadTimeMs: 2500,
		TotalBytes: 1024 * 500, // 500 KB
		Credit:     "loadshow",
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
	if bounds.Dx() != input.Width {
		t.Errorf("expected image width %d, got %d", input.Width, bounds.Dx())
	}

	// Check that CaptureHTMLWithViewport was called
	if len(mockCapturer.CaptureHTMLWithViewportCalls) != 1 {
		t.Errorf("expected 1 call to CaptureHTMLWithViewport, got %d",
			len(mockCapturer.CaptureHTMLWithViewportCalls))
	}
}

func TestStage_Execute_WithDebugSink(t *testing.T) {
	mockCapturer := mocks.NewHTMLCapturer()
	mockSink := mocks.NewDebugSink(true)

	stage := NewStage(mockCapturer, mockSink)

	input := pipeline.BannerInput{
		Width:      400,
		Height:     80,
		URL:        "https://example.com",
		Title:      "Test",
		LoadTimeMs: 1000,
		TotalBytes: 1024,
		Credit:     "test",
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

func TestStage_Execute_CustomCredit(t *testing.T) {
	mockCapturer := mocks.NewHTMLCapturer()
	mockSink := mocks.NewDebugSink(false)

	stage := NewStage(mockCapturer, mockSink)

	input := pipeline.BannerInput{
		Width:      400,
		Height:     80,
		URL:        "https://example.com",
		Title:      "Test",
		LoadTimeMs: 1000,
		TotalBytes: 1024,
		Credit:     "Custom Credit",
		Theme:      pipeline.DefaultBannerTheme(),
	}

	_, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that CaptureHTMLWithViewport was called with HTML containing the credit
	if len(mockCapturer.CaptureHTMLWithViewportCalls) != 1 {
		t.Fatalf("expected 1 call to CaptureHTMLWithViewport, got %d",
			len(mockCapturer.CaptureHTMLWithViewportCalls))
	}

	// The HTML should contain the custom credit
	html := mockCapturer.CaptureHTMLWithViewportCalls[0].HTML
	if !contains(html, "Custom Credit") {
		t.Error("expected HTML to contain custom credit text")
	}
}

func TestNewTemplateVars(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		url        string
		title      string
		loadTimeMs int
		totalBytes int64
		credit     string
		wantCredit string
	}{
		{
			name:       "with custom credit",
			width:      400,
			url:        "https://example.com",
			title:      "Test",
			loadTimeMs: 1000,
			totalBytes: 1024,
			credit:     "My Credit",
			wantCredit: "My Credit",
		},
		{
			name:       "with empty credit defaults to loadshow",
			width:      400,
			url:        "https://example.com",
			title:      "Test",
			loadTimeMs: 1000,
			totalBytes: 1024,
			credit:     "",
			wantCredit: "loadshow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := NewTemplateVars(tt.width, tt.url, tt.title, tt.loadTimeMs, tt.totalBytes, tt.credit)
			if vars.Credit != tt.wantCredit {
				t.Errorf("Credit = %q, want %q", vars.Credit, tt.wantCredit)
			}
			if vars.BodyWidth != tt.width {
				t.Errorf("BodyWidth = %d, want %d", vars.BodyWidth, tt.width)
			}
			if vars.MainTitle != tt.title {
				t.Errorf("MainTitle = %q, want %q", vars.MainTitle, tt.title)
			}
			if vars.SubTitle != tt.url {
				t.Errorf("SubTitle = %q, want %q", vars.SubTitle, tt.url)
			}
		})
	}
}

func TestRenderHTML(t *testing.T) {
	vars := NewTemplateVars(400, "https://example.com", "Test Title", 2500, 1024*1024, "loadshow")

	html, err := RenderHTML(vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that HTML contains expected elements
	checks := []string{
		"Test Title",
		"https://example.com",
		"loadshow",
		"Traffic",
		"OnLoad Time",
	}

	for _, check := range checks {
		if !contains(html, check) {
			t.Errorf("expected HTML to contain %q", check)
		}
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
