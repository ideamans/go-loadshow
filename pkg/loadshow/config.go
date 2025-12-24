// Package loadshow provides a high-level API for creating page load videos.
package loadshow

import (
	"image/color"

	"github.com/user/loadshow/pkg/orchestrator"
	"github.com/user/loadshow/pkg/ports"
)

// Config represents the configuration for loadshow video generation.
type Config struct {
	// Video size
	Width  int // Output video width (default: 512)
	Height int // Output video height (default: 640, includes banner)

	// Layout
	ViewportWidth int // Browser viewport width (min: 500)
	Columns       int // Number of columns (min: 1)
	Margin        int // Margin around the canvas (top, bottom, left, right)
	Gap           int // Gap between columns
	Indent        int // Additional top margin for columns 2+
	Outdent       int // Additional bottom margin for column 1

	// Style
	BackgroundColor color.Color // Canvas background color
	BorderColor     color.Color // Column border color
	BorderWidth     int         // Border width in pixels

	// Encoding
	Quality int // MP4 quality (CRF 0-63, lower is better)
	OutroMs int // Duration to hold final frame in milliseconds

	// Banner
	Credit string // Text shown in banner (replaces "loadshow")

	// Network throttling
	DownloadSpeed int // Download speed in bytes/sec (0 = unlimited)
	UploadSpeed   int // Upload speed in bytes/sec (0 = unlimited)

	// CPU throttling
	CPUThrottling float64 // CPU slowdown factor (1.0 = no throttling, 4.0 = 4x slower)

	// Browser options
	IgnoreHTTPSErrors bool   // Ignore HTTPS certificate errors
	ProxyServer       string // HTTP proxy server (e.g., "http://proxy:8080")
}

// ConfigBuilder provides a fluent interface for building Config.
type ConfigBuilder struct {
	config Config
}

// NewConfigBuilder creates a new ConfigBuilder with desktop preset defaults.
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: desktopDefaults(),
	}
}

// NewMobileConfigBuilder creates a new ConfigBuilder with mobile preset defaults.
func NewMobileConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: mobileDefaults(),
	}
}

// desktopDefaults returns the desktop preset configuration.
func desktopDefaults() Config {
	return Config{
		// Video size
		Width:  512,
		Height: 640,

		// Layout
		ViewportWidth: 1024,
		Columns:       2,
		Margin:        20,
		Gap:           20,
		Indent:        20,
		Outdent:       20,

		// Style
		BackgroundColor: color.RGBA{R: 220, G: 220, B: 220, A: 255}, // #dcdcdc
		BorderColor:     color.RGBA{R: 180, G: 180, B: 180, A: 255}, // #b4b4b4
		BorderWidth:     1,

		// Encoding
		Quality: 30,
		OutroMs: 2000,

		// Banner
		Credit: "loadshow",

		// Network (no throttling)
		DownloadSpeed: 0,
		UploadSpeed:   0,

		// CPU (no throttling)
		CPUThrottling: 1.0,
	}
}

// mobileDefaults returns the mobile preset configuration.
func mobileDefaults() Config {
	return Config{
		// Video size
		Width:  512,
		Height: 640,

		// Layout
		ViewportWidth: 500,
		Columns:       3,
		Margin:        20,
		Gap:           20,
		Indent:        20,
		Outdent:       20,

		// Style
		BackgroundColor: color.RGBA{R: 220, G: 220, B: 220, A: 255}, // #dcdcdc
		BorderColor:     color.RGBA{R: 180, G: 180, B: 180, A: 255}, // #b4b4b4
		BorderWidth:     1,

		// Encoding
		Quality: 30,
		OutroMs: 2000,

		// Banner
		Credit: "loadshow",

		// Network (10 Mbps = 10 * 1024 * 1024 / 8 bytes/sec)
		DownloadSpeed: 10 * 1024 * 1024 / 8, // 1,310,720 bytes/sec
		UploadSpeed:   10 * 1024 * 1024 / 8,

		// CPU (4x slower)
		CPUThrottling: 4.0,
	}
}

// Build returns the final Config, applying validation and constraints.
func (b *ConfigBuilder) Build() Config {
	cfg := b.config

	// Enforce minimum viewport width of 500
	if cfg.ViewportWidth < 500 {
		cfg.ViewportWidth = 500
	}

	// Enforce minimum columns of 1
	if cfg.Columns < 1 {
		cfg.Columns = 1
	}

	return cfg
}

// WithWidth sets the output video width.
func (b *ConfigBuilder) WithWidth(width int) *ConfigBuilder {
	b.config.Width = width
	return b
}

// WithHeight sets the output video height (includes banner).
func (b *ConfigBuilder) WithHeight(height int) *ConfigBuilder {
	b.config.Height = height
	return b
}

// WithViewportWidth sets the browser viewport width.
// Values below 500 will be forced to 500.
func (b *ConfigBuilder) WithViewportWidth(width int) *ConfigBuilder {
	b.config.ViewportWidth = width
	return b
}

// WithColumns sets the number of columns.
// Values below 1 will be forced to 1.
func (b *ConfigBuilder) WithColumns(columns int) *ConfigBuilder {
	b.config.Columns = columns
	return b
}

// WithMargin sets the margin around the canvas.
func (b *ConfigBuilder) WithMargin(margin int) *ConfigBuilder {
	b.config.Margin = margin
	return b
}

// WithGap sets the gap between columns.
func (b *ConfigBuilder) WithGap(gap int) *ConfigBuilder {
	b.config.Gap = gap
	return b
}

// WithIndent sets the additional top margin for columns 2+.
func (b *ConfigBuilder) WithIndent(indent int) *ConfigBuilder {
	b.config.Indent = indent
	return b
}

// WithOutdent sets the additional bottom margin for column 1.
func (b *ConfigBuilder) WithOutdent(outdent int) *ConfigBuilder {
	b.config.Outdent = outdent
	return b
}

// WithBackgroundColor sets the canvas background color.
func (b *ConfigBuilder) WithBackgroundColor(c color.Color) *ConfigBuilder {
	b.config.BackgroundColor = c
	return b
}

// WithBorderColor sets the column border color.
func (b *ConfigBuilder) WithBorderColor(c color.Color) *ConfigBuilder {
	b.config.BorderColor = c
	return b
}

// WithBorderWidth sets the border width in pixels.
func (b *ConfigBuilder) WithBorderWidth(width int) *ConfigBuilder {
	b.config.BorderWidth = width
	return b
}

// WithQuality sets the MP4 quality (CRF 0-63, lower is better).
func (b *ConfigBuilder) WithQuality(quality int) *ConfigBuilder {
	b.config.Quality = quality
	return b
}

// WithOutroMs sets the duration to hold the final frame in milliseconds.
func (b *ConfigBuilder) WithOutroMs(ms int) *ConfigBuilder {
	b.config.OutroMs = ms
	return b
}

// WithCredit sets the text shown in the banner.
func (b *ConfigBuilder) WithCredit(credit string) *ConfigBuilder {
	b.config.Credit = credit
	return b
}

// WithDownloadSpeed sets the download speed limit in bytes/sec.
// Use 0 for unlimited.
func (b *ConfigBuilder) WithDownloadSpeed(bytesPerSec int) *ConfigBuilder {
	b.config.DownloadSpeed = bytesPerSec
	return b
}

// WithUploadSpeed sets the upload speed limit in bytes/sec.
// Use 0 for unlimited.
func (b *ConfigBuilder) WithUploadSpeed(bytesPerSec int) *ConfigBuilder {
	b.config.UploadSpeed = bytesPerSec
	return b
}

// WithNetworkSpeed sets both download and upload speed limits in bytes/sec.
// Use 0 for unlimited.
func (b *ConfigBuilder) WithNetworkSpeed(bytesPerSec int) *ConfigBuilder {
	b.config.DownloadSpeed = bytesPerSec
	b.config.UploadSpeed = bytesPerSec
	return b
}

// WithCPUThrottling sets the CPU slowdown factor.
// 1.0 = no throttling, 4.0 = 4x slower.
func (b *ConfigBuilder) WithCPUThrottling(factor float64) *ConfigBuilder {
	b.config.CPUThrottling = factor
	return b
}

// WithIgnoreHTTPSErrors enables ignoring HTTPS certificate errors.
func (b *ConfigBuilder) WithIgnoreHTTPSErrors(ignore bool) *ConfigBuilder {
	b.config.IgnoreHTTPSErrors = ignore
	return b
}

// WithProxyServer sets the HTTP proxy server.
func (b *ConfigBuilder) WithProxyServer(proxy string) *ConfigBuilder {
	b.config.ProxyServer = proxy
	return b
}

// Mbps converts megabits per second to bytes per second.
// Useful for setting network speeds: WithDownloadSpeed(Mbps(10))
func Mbps(mbps int) int {
	return mbps * 1024 * 1024 / 8
}

// ToOrchestratorConfig converts Config to orchestrator.Config.
// Width/Height define the video dimensions; layout is computed from these.
func (c Config) ToOrchestratorConfig(url, outputPath string) orchestrator.Config {
	return orchestrator.Config{
		URL:        url,
		OutputPath: outputPath,

		// Layout - use Width/Height directly
		CanvasWidth:    c.Width,
		CanvasHeight:   c.Height,
		Columns:        c.Columns,
		Gap:            c.Gap,
		Padding:        c.Margin,
		BorderWidth:    c.BorderWidth,
		Indent:         c.Indent,
		Outdent:        c.Outdent,
		ProgressHeight: 16,

		// Style
		BackgroundColor: colorToArray(c.BackgroundColor),
		BorderColor:     colorToArray(c.BorderColor),

		// Recording
		ViewportWidth: c.ViewportWidth,
		TimeoutMs:     30000,
		NetworkConditions: ports.NetworkConditions{
			DownloadSpeed: c.DownloadSpeed,
			UploadSpeed:   c.UploadSpeed,
		},
		CPUThrottling: c.CPUThrottling,

		// Browser options
		IgnoreHTTPSErrors: c.IgnoreHTTPSErrors,
		ProxyServer:       c.ProxyServer,

		// Banner
		BannerEnabled: true,
		BannerHeight:  80,
		Credit:        c.Credit,

		// Composition
		ShowProgress: true,

		// Encoding
		Quality: c.Quality,
		Bitrate: 0,
		OutroMs: c.OutroMs,
		FPS:     30.0,
	}
}

// colorToArray converts color.Color to [4]uint8 array.
func colorToArray(c color.Color) [4]uint8 {
	r, g, b, a := c.RGBA()
	return [4]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}
