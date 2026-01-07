package summarizer

import (
	"testing"
	"time"
)

func TestNewSummary(t *testing.T) {
	before := time.Now()
	summary := NewSummary()
	after := time.Now()

	if summary.GeneratedAt.Before(before) || summary.GeneratedAt.After(after) {
		t.Errorf("GeneratedAt should be between %v and %v, got %v",
			before, after, summary.GeneratedAt)
	}
}

func TestBuilder_WithPage(t *testing.T) {
	summary := NewBuilder().
		WithPage("Test Title", "https://example.com").
		Build()

	if summary.Page.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got '%s'", summary.Page.Title)
	}
	if summary.Page.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got '%s'", summary.Page.URL)
	}
}

func TestBuilder_WithTiming(t *testing.T) {
	summary := NewBuilder().
		WithTiming(1000, 2000, 3000).
		Build()

	if summary.Timing.DOMContentLoadedMs != 1000 {
		t.Errorf("expected DOMContentLoadedMs 1000, got %d", summary.Timing.DOMContentLoadedMs)
	}
	if summary.Timing.LoadCompleteMs != 2000 {
		t.Errorf("expected LoadCompleteMs 2000, got %d", summary.Timing.LoadCompleteMs)
	}
	if summary.Timing.TotalDurationMs != 3000 {
		t.Errorf("expected TotalDurationMs 3000, got %d", summary.Timing.TotalDurationMs)
	}
}

func TestBuilder_WithTimeout(t *testing.T) {
	summary := NewBuilder().
		WithTiming(1000, 2000, 3000).
		WithTimeout(true, 30).
		Build()

	if !summary.Timing.TimedOut {
		t.Error("expected TimedOut to be true")
	}
	if summary.Timing.TimeoutSec != 30 {
		t.Errorf("expected TimeoutSec 30, got %d", summary.Timing.TimeoutSec)
	}
}

func TestBuilder_WithTraffic(t *testing.T) {
	summary := NewBuilder().
		WithTraffic(1024 * 1024).
		Build()

	if summary.Traffic.TotalBytes != 1024*1024 {
		t.Errorf("expected TotalBytes %d, got %d", 1024*1024, summary.Traffic.TotalBytes)
	}
}

func TestBuilder_WithSettings(t *testing.T) {
	settings := Settings{
		Preset:        "mobile",
		Quality:       "medium",
		Codec:         "H.264",
		ViewportWidth: 375,
		Columns:       3,
		DownloadSpeed: 1310720,
		UploadSpeed:   655360,
		CPUThrottling: 4.0,
	}

	summary := NewBuilder().
		WithSettings(settings).
		Build()

	if summary.Settings.Preset != "mobile" {
		t.Errorf("expected Preset 'mobile', got '%s'", summary.Settings.Preset)
	}
	if summary.Settings.Columns != 3 {
		t.Errorf("expected Columns 3, got %d", summary.Settings.Columns)
	}
	if summary.Settings.CPUThrottling != 4.0 {
		t.Errorf("expected CPUThrottling 4.0, got %f", summary.Settings.CPUThrottling)
	}
}

func TestBuilder_WithVideo(t *testing.T) {
	video := VideoInfo{
		FrameCount:    100,
		DurationMs:    5000,
		FileSize:      102400,
		CanvasWidth:   512,
		CanvasHeight:  640,
		CRF:           25,
		OutroDuration: 2000,
	}

	summary := NewBuilder().
		WithVideo(video).
		Build()

	if summary.Video.FrameCount != 100 {
		t.Errorf("expected FrameCount 100, got %d", summary.Video.FrameCount)
	}
	if summary.Video.FileSize != 102400 {
		t.Errorf("expected FileSize 102400, got %d", summary.Video.FileSize)
	}
}

func TestBuilder_FullChain(t *testing.T) {
	summary := NewBuilder().
		WithPage("Test Page", "https://example.com").
		WithTiming(1000, 2000, 3000).
		WithTimeout(false, 30).
		WithTraffic(1024 * 1024).
		WithSettings(Settings{
			Preset:  "mobile",
			Quality: "medium",
		}).
		WithVideo(VideoInfo{
			FrameCount: 50,
		}).
		Build()

	// Verify all fields are set
	if summary.Page.Title != "Test Page" {
		t.Error("Page.Title not set correctly")
	}
	if summary.Timing.DOMContentLoadedMs != 1000 {
		t.Error("Timing.DOMContentLoadedMs not set correctly")
	}
	if summary.Timing.TimedOut {
		t.Error("Timing.TimedOut should be false")
	}
	if summary.Traffic.TotalBytes != 1024*1024 {
		t.Error("Traffic.TotalBytes not set correctly")
	}
	if summary.Settings.Preset != "mobile" {
		t.Error("Settings.Preset not set correctly")
	}
	if summary.Video.FrameCount != 50 {
		t.Error("Video.FrameCount not set correctly")
	}
}
