// Package summarizer provides summary generation for recording results.
package summarizer

import "time"

// Summary contains all data collected during a recording session.
type Summary struct {
	// Metadata
	GeneratedAt time.Time

	// Page information
	Page PageInfo

	// Timing results
	Timing TimingInfo

	// Traffic information
	Traffic TrafficInfo

	// Recording settings
	Settings Settings

	// Video output details
	Video VideoInfo
}

// PageInfo contains information about the recorded page.
type PageInfo struct {
	Title string
	URL   string
}

// TimingInfo contains timing measurements.
type TimingInfo struct {
	DOMContentLoadedMs int
	LoadCompleteMs     int
	TotalDurationMs    int
}

// TrafficInfo contains network traffic information.
type TrafficInfo struct {
	TotalBytes int64
}

// Settings contains the recording configuration.
type Settings struct {
	Preset        string
	Quality       string
	Codec         string
	ViewportWidth int
	Columns       int

	// Network throttling (bytes/sec, 0 = unlimited)
	DownloadSpeed int
	UploadSpeed   int

	// CPU throttling (1.0 = no throttling)
	CPUThrottling float64
}

// VideoInfo contains information about the output video.
type VideoInfo struct {
	FrameCount    int
	DurationMs    int
	FileSize      int64
	CanvasWidth   int
	CanvasHeight  int
	CRF           int
	OutroDuration int // ms
}

// NewSummary creates a new Summary with the current timestamp.
func NewSummary() *Summary {
	return &Summary{
		GeneratedAt: time.Now(),
	}
}

// Builder provides a fluent interface for building a Summary.
type Builder struct {
	summary *Summary
}

// NewBuilder creates a new Builder.
func NewBuilder() *Builder {
	return &Builder{
		summary: NewSummary(),
	}
}

// WithPage sets page information.
func (b *Builder) WithPage(title, url string) *Builder {
	b.summary.Page = PageInfo{
		Title: title,
		URL:   url,
	}
	return b
}

// WithTiming sets timing information.
func (b *Builder) WithTiming(domContentLoaded, loadComplete, totalDuration int) *Builder {
	b.summary.Timing = TimingInfo{
		DOMContentLoadedMs: domContentLoaded,
		LoadCompleteMs:     loadComplete,
		TotalDurationMs:    totalDuration,
	}
	return b
}

// WithTraffic sets traffic information.
func (b *Builder) WithTraffic(totalBytes int64) *Builder {
	b.summary.Traffic = TrafficInfo{
		TotalBytes: totalBytes,
	}
	return b
}

// WithSettings sets recording settings.
func (b *Builder) WithSettings(settings Settings) *Builder {
	b.summary.Settings = settings
	return b
}

// WithVideo sets video output information.
func (b *Builder) WithVideo(video VideoInfo) *Builder {
	b.summary.Video = video
	return b
}

// Build returns the constructed Summary.
func (b *Builder) Build() *Summary {
	return b.summary
}
