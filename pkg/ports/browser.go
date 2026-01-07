// Package ports defines interfaces for external dependencies.
package ports

import (
	"context"
)

// Browser abstracts browser automation for page recording.
type Browser interface {
	// Launch starts the browser with the given options.
	Launch(ctx context.Context, opts BrowserOptions) error

	// Navigate loads the specified URL.
	Navigate(url string) error

	// SetViewport sets the browser viewport dimensions with device scale factor.
	// viewportWidth/viewportHeight are in CSS pixels.
	// screenWidth/screenHeight are in device pixels (used for screencast).
	// deviceScaleFactor determines how CSS pixels map to device pixels.
	SetViewport(viewportWidth, viewportHeight, screenWidth, screenHeight int, deviceScaleFactor float64) error

	// SetNetworkConditions configures network throttling.
	SetNetworkConditions(conditions NetworkConditions) error

	// SetCPUThrottling sets CPU throttling rate (e.g., 4.0 means 4x slower).
	SetCPUThrottling(rate float64) error

	// StartScreencast begins capturing screenshots at regular intervals.
	// maxWidth/maxHeight constrain the output image dimensions.
	// Returns a channel that receives screen frames.
	StartScreencast(quality, maxWidth, maxHeight int) (<-chan ScreenFrame, error)

	// StopScreencast stops the screencast capture.
	StopScreencast() error

	// GetPageInfo retrieves information about the current page.
	GetPageInfo() (*PageInfo, error)

	// GetPerformanceTiming retrieves navigation timing metrics.
	GetPerformanceTiming() (*PerformanceTiming, error)

	// Close shuts down the browser.
	Close() error
}

// BrowserOptions configures browser launch settings.
type BrowserOptions struct {
	Headless          bool
	ChromePath        string
	UserAgent         string
	Headers           map[string]string
	WindowWidth       int    // Initial window width (for screencast)
	WindowHeight      int    // Initial window height (for screencast)
	IgnoreHTTPSErrors bool   // Ignore HTTPS certificate errors
	ProxyServer       string // HTTP proxy server (e.g., "http://proxy:8080")
	Incognito         bool   // Run browser in incognito mode (default: true)
}

// NetworkConditions defines network throttling parameters.
type NetworkConditions struct {
	LatencyMs     int  // Network latency in milliseconds
	DownloadSpeed int  // Download speed in bytes/sec
	UploadSpeed   int  // Upload speed in bytes/sec
	Offline       bool // Whether to simulate offline mode
}

// ScreenFrame represents a single captured screenshot.
type ScreenFrame struct {
	TimestampMs int    // Timestamp in milliseconds since navigation start
	Data        []byte // JPEG image data
	Metadata    ScreenFrameMetadata
}

// ScreenFrameMetadata contains additional information about a frame.
type ScreenFrameMetadata struct {
	LoadedResources int   // Number of resources loaded at this point
	TotalResources  int   // Total resources being loaded
	TotalBytes      int64 // Total bytes transferred
}

// PageInfo contains information about the current page.
type PageInfo struct {
	Title        string
	URL          string
	ScrollHeight int
	ScrollWidth  int
}

// PerformanceTiming contains navigation timing metrics from Performance API.
type PerformanceTiming struct {
	NavigationStart      int64 // When navigation started
	DOMContentLoadedEnd  int64 // When DOMContentLoaded event completed
	LoadEventEnd         int64 // When load event completed
}
