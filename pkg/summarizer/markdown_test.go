package summarizer

import (
	"strings"
	"testing"
	"time"
)

func TestMarkdownFormatter_Format_Basic(t *testing.T) {
	formatter := NewMarkdownFormatter()

	summary := &Summary{
		GeneratedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Page: PageInfo{
			Title: "Test Page",
			URL:   "https://example.com",
		},
		Timing: TimingInfo{
			DOMContentLoadedMs: 1000,
			LoadCompleteMs:     2000,
			TotalDurationMs:    3000,
			TimedOut:           false,
			TimeoutSec:         30,
		},
		Traffic: TrafficInfo{
			TotalBytes: 1024 * 1024, // 1 MB
		},
		Settings: Settings{
			Preset:        "mobile",
			Quality:       "medium",
			Codec:         "H.264",
			ViewportWidth: 375,
			Columns:       3,
			DownloadSpeed: 1310720,
			UploadSpeed:   655360,
			CPUThrottling: 4.0,
		},
		Video: VideoInfo{
			FrameCount:    100,
			DurationMs:    5000,
			FileSize:      102400,
			CanvasWidth:   512,
			CanvasHeight:  640,
			CRF:           25,
			OutroDuration: 2000,
		},
	}

	result := formatter.Format(summary)

	// Check required sections
	checks := []string{
		"# Recording Summary",
		"Test Page",
		"https://example.com",
		"1000 ms",  // DCL
		"2000 ms",  // Load Complete
		"1.00 MB",  // Traffic
		"mobile",   // Preset
		"medium",   // Quality
		"H.264",    // Codec
		"100",      // Frame count
		"512x640",  // Canvas size
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("expected output to contain %q", check)
		}
	}
}

func TestMarkdownFormatter_Format_WithTimeout(t *testing.T) {
	formatter := NewMarkdownFormatter()

	summary := &Summary{
		GeneratedAt: time.Now(),
		Page: PageInfo{
			Title: "Test Page",
			URL:   "https://example.com",
		},
		Timing: TimingInfo{
			DOMContentLoadedMs: 500,
			LoadCompleteMs:     0,
			TotalDurationMs:    1000,
			TimedOut:           true,
			TimeoutSec:         1,
		},
		Traffic: TrafficInfo{
			TotalBytes: 1024,
		},
	}

	result := formatter.Format(summary)

	// Should show timeout for Load Complete
	if !strings.Contains(result, "Timeout") {
		t.Error("expected output to contain 'Timeout' for Load Complete")
	}
	if !strings.Contains(result, "(1s)") {
		t.Error("expected output to contain timeout seconds '(1s)'")
	}

	// DCL should still show value since it's within timeout
	if !strings.Contains(result, "500 ms") {
		t.Error("expected output to contain DCL value '500 ms'")
	}
}

func TestMarkdownFormatter_Format_DCL_NA(t *testing.T) {
	tests := []struct {
		name     string
		timing   TimingInfo
		wantNA   bool
		wantDCL  string
	}{
		{
			name: "DCL is 0",
			timing: TimingInfo{
				DOMContentLoadedMs: 0,
				TimedOut:           true,
				TimeoutSec:         1,
			},
			wantNA: true,
		},
		{
			name: "DCL exceeds timeout",
			timing: TimingInfo{
				DOMContentLoadedMs: 2000, // 2 seconds
				TimedOut:           true,
				TimeoutSec:         1, // 1 second timeout
			},
			wantNA: true,
		},
		{
			name: "DCL within timeout",
			timing: TimingInfo{
				DOMContentLoadedMs: 500,
				TimedOut:           true,
				TimeoutSec:         1,
			},
			wantNA:  false,
			wantDCL: "500 ms",
		},
		{
			name: "No timeout, normal DCL",
			timing: TimingInfo{
				DOMContentLoadedMs: 1500,
				TimedOut:           false,
				TimeoutSec:         30,
			},
			wantNA:  false,
			wantDCL: "1500 ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewMarkdownFormatter()
			summary := &Summary{
				GeneratedAt: time.Now(),
				Page:        PageInfo{Title: "Test", URL: "https://example.com"},
				Timing:      tt.timing,
			}

			result := formatter.Format(summary)

			if tt.wantNA {
				if !strings.Contains(result, "N/A") {
					t.Error("expected output to contain 'N/A' for DCL")
				}
			} else {
				if strings.Contains(result, "N/A") {
					t.Error("expected output NOT to contain 'N/A' for DCL")
				}
				if !strings.Contains(result, tt.wantDCL) {
					t.Errorf("expected output to contain '%s'", tt.wantDCL)
				}
			}
		})
	}
}

func TestMarkdownFormatter_WithTranslator(t *testing.T) {
	translator := func(key string) string {
		translations := map[string]string{
			"Recording Summary": "記録サマリー",
			"Page Title":        "ページタイトル",
			"Timeout":           "タイムアウト",
		}
		if v, ok := translations[key]; ok {
			return v
		}
		return key
	}

	formatter := NewMarkdownFormatter(WithTranslator(translator))

	summary := &Summary{
		GeneratedAt: time.Now(),
		Page:        PageInfo{Title: "Test", URL: "https://example.com"},
		Timing: TimingInfo{
			TimedOut:   true,
			TimeoutSec: 5,
		},
	}

	result := formatter.Format(summary)

	if !strings.Contains(result, "記録サマリー") {
		t.Error("expected translated 'Recording Summary'")
	}
	if !strings.Contains(result, "ページタイトル") {
		t.Error("expected translated 'Page Title'")
	}
	if !strings.Contains(result, "タイムアウト") {
		t.Error("expected translated 'Timeout'")
	}
}

func TestMarkdownFormatter_WithVersion(t *testing.T) {
	formatter := NewMarkdownFormatter(WithVersion("v1.2.0"))

	summary := &Summary{
		GeneratedAt: time.Now(),
		Page:        PageInfo{Title: "Test", URL: "https://example.com"},
	}

	result := formatter.Format(summary)

	if !strings.Contains(result, "v1.2.0") {
		t.Error("expected output to contain version 'v1.2.0'")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1536 * 1024 * 1024, "1.50 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestMarkdownFormatter_NoTotalDuration(t *testing.T) {
	formatter := NewMarkdownFormatter()

	summary := &Summary{
		GeneratedAt: time.Now(),
		Page:        PageInfo{Title: "Test", URL: "https://example.com"},
		Timing: TimingInfo{
			DOMContentLoadedMs: 1000,
			LoadCompleteMs:     2000,
			TotalDurationMs:    5000, // This should NOT appear in output
		},
	}

	result := formatter.Format(summary)

	// Total Duration should not be in the output
	if strings.Contains(result, "Total Duration") {
		t.Error("output should NOT contain 'Total Duration'")
	}
	if strings.Contains(result, "5000 ms") {
		t.Error("output should NOT contain total duration value")
	}
}
