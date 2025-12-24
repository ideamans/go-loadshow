package pipeline

import (
	"image"
	"image/color"

	"github.com/user/loadshow/pkg/ports"
)

// =============================================================================
// Common Types
// =============================================================================

// Dimension represents width and height.
type Dimension struct {
	Width  int
	Height int
}

// Rectangle represents a rectangular area.
type Rectangle struct {
	X      int
	Y      int
	Width  int
	Height int
}

// =============================================================================
// Layout Stage Types
// =============================================================================

// LayoutInput contains parameters for layout calculation.
type LayoutInput struct {
	CanvasWidth    int // Total canvas width (default: 512)
	CanvasHeight   int // Total canvas height (default: 640)
	Columns        int // Number of columns (default: 3)
	Gap            int // Gap between columns (default: 20)
	Padding        int // Padding around the canvas (default: 20)
	BorderWidth    int // Border width for column frames (default: 1)
	Indent         int // Indent for non-first columns (default: 20)
	Outdent        int // Outdent for first column (default: 20)
	BannerHeight   int // Height of banner area (default: 0)
	ProgressHeight int // Height of progress bar (default: 16)
}

// DefaultLayoutInput returns LayoutInput with default values.
func DefaultLayoutInput() LayoutInput {
	return LayoutInput{
		CanvasWidth:    512,
		CanvasHeight:   640,
		Columns:        3,
		Gap:            20,
		Padding:        20,
		BorderWidth:    1,
		Indent:         20,
		Outdent:        20,
		BannerHeight:   0,
		ProgressHeight: 16,
	}
}

// LayoutResult contains the calculated layout dimensions and positions.
type LayoutResult struct {
	// Scroll represents the total scrollable area dimensions.
	// This is the viewport size that the browser should use.
	Scroll Dimension

	// Columns contains the rectangles for each column frame (for decoration).
	Columns []Rectangle

	// Windows contains the content display areas with scroll positions.
	Windows []Window

	// BannerArea is the rectangle for the banner (if enabled).
	BannerArea Rectangle

	// ProgressArea is the rectangle for the progress bar.
	ProgressArea Rectangle

	// ContentArea is the main content area.
	ContentArea Rectangle
}

// Window represents a viewport window with scroll position.
type Window struct {
	Rectangle
	ScrollTop int // Vertical scroll position for this window
}

// =============================================================================
// Record Stage Types
// =============================================================================

// RecordInput contains parameters for page recording.
type RecordInput struct {
	URL               string
	ViewportWidth     int    // Browser viewport width (e.g., 375 for mobile)
	Screen            Dimension // Target screen dimensions from layout (scroll width/height)
	TimeoutMs         int
	NetworkConditions ports.NetworkConditions
	CPUThrottling     float64
	Headers           map[string]string
}

// DefaultRecordInput returns RecordInput with default values.
func DefaultRecordInput() RecordInput {
	return RecordInput{
		ViewportWidth: 375,
		Screen:        Dimension{Width: 144, Height: 1739}, // Default for 3-column layout
		TimeoutMs:     30000,
		NetworkConditions: ports.NetworkConditions{
			LatencyMs:     20,
			DownloadSpeed: 10 * 1024 * 1024 / 8, // 10 Mbps in bytes/sec
			UploadSpeed:   5 * 1024 * 1024 / 8,  // 5 Mbps in bytes/sec
		},
		CPUThrottling: 4.0,
	}
}

// RecordResult contains the recording output.
type RecordResult struct {
	Frames   []RawFrame
	PageInfo ports.PageInfo
	Timing   TimingInfo
}

// RawFrame represents a single recorded frame.
type RawFrame struct {
	TimestampMs     int    // Timestamp in milliseconds since navigation start
	ImageData       []byte // JPEG image data
	LoadedResources int    // Number of resources loaded at this point
	TotalResources  int    // Total resources being loaded
	TotalBytes      int64  // Total bytes transferred
}

// TimingInfo contains page load timing information.
type TimingInfo struct {
	NavigationStartMs  int
	DOMContentLoadedMs int
	LoadCompleteMs     int
	TotalDurationMs    int
}

// =============================================================================
// Banner Stage Types
// =============================================================================

// BannerInput contains parameters for banner generation.
type BannerInput struct {
	Width      int
	Height     int
	URL        string
	Title      string
	LoadTimeMs int
	TotalBytes int64
	Theme      BannerTheme
}

// BannerTheme defines banner styling.
type BannerTheme struct {
	BackgroundColor color.Color
	TextColor       color.Color
	AccentColor     color.Color
}

// DefaultBannerTheme returns a default banner theme.
func DefaultBannerTheme() BannerTheme {
	return BannerTheme{
		BackgroundColor: color.RGBA{R: 45, G: 45, B: 45, A: 255},
		TextColor:       color.White,
		AccentColor:     color.RGBA{R: 100, G: 180, B: 255, A: 255},
	}
}

// BannerResult contains the generated banner.
type BannerResult struct {
	Image image.Image
}

// =============================================================================
// Composite Stage Types
// =============================================================================

// CompositeInput contains parameters for frame composition.
type CompositeInput struct {
	RawFrames    []RawFrame
	Layout       LayoutResult
	Banner       *BannerResult // Optional banner
	Theme        CompositeTheme
	ShowProgress bool
	TotalTimeMs  int // Total recording time for progress calculation
}

// CompositeTheme defines composition styling.
type CompositeTheme struct {
	BackgroundColor  color.Color
	BorderColor      color.Color
	ProgressBarColor color.Color
	ProgressBgColor  color.Color
}

// DefaultCompositeTheme returns a default composite theme.
func DefaultCompositeTheme() CompositeTheme {
	return CompositeTheme{
		BackgroundColor:  color.RGBA{R: 30, G: 30, B: 30, A: 255},
		BorderColor:      color.RGBA{R: 80, G: 80, B: 80, A: 255},
		ProgressBarColor: color.RGBA{R: 76, G: 175, B: 80, A: 255},
		ProgressBgColor:  color.RGBA{R: 60, G: 60, B: 60, A: 255},
	}
}

// CompositeResult contains the composed frames.
type CompositeResult struct {
	Frames []ComposedFrame
}

// ComposedFrame represents a fully composed frame.
type ComposedFrame struct {
	TimestampMs int
	Image       image.Image
}

// =============================================================================
// Encode Stage Types
// =============================================================================

// EncodeInput contains parameters for video encoding.
type EncodeInput struct {
	Frames  []ComposedFrame
	OutroMs int     // Duration to hold the last frame
	Quality int     // CRF: 0-63 (lower is higher quality)
	Bitrate int     // Target bitrate in kbps
	FPS     float64 // Frames per second
}

// DefaultEncodeInput returns EncodeInput with default values.
func DefaultEncodeInput() EncodeInput {
	return EncodeInput{
		OutroMs: 1000,
		Quality: 30,
		Bitrate: 1000,
		FPS:     30.0,
	}
}

// EncodeResult contains the encoded video.
type EncodeResult struct {
	VideoData  []byte
	DurationMs int
	FileSize   int64
}
