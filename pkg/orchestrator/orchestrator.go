// Package orchestrator coordinates all pipeline stages.
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"

	"github.com/ideamans/go-l10n"
	"github.com/user/loadshow/pkg/pipeline"
	"github.com/user/loadshow/pkg/ports"
)

// Config contains all configuration for the orchestrator.
type Config struct {
	// Input
	URL        string
	OutputPath string

	// Layout
	CanvasWidth    int
	CanvasHeight   int
	Columns        int
	Gap            int
	Padding        int
	BorderWidth    int
	Indent         int
	Outdent        int
	ProgressHeight int

	// Style
	BackgroundColor [4]uint8 // RGBA
	BorderColor     [4]uint8 // RGBA

	// Recording
	ViewportWidth     int
	ScreencastQuality int // JPEG quality for screencast (0-100)
	TimeoutMs         int
	NetworkConditions ports.NetworkConditions
	CPUThrottling     float64
	Headers           map[string]string

	// Browser options
	IgnoreHTTPSErrors bool
	ProxyServer       string

	// Banner
	BannerEnabled bool
	BannerHeight  int
	Credit        string // Banner credit text

	// Composition
	ShowProgress bool

	// Encoding
	VideoCRF int
	Bitrate  int
	OutroMs  int
	FPS      float64
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		CanvasWidth:    512,
		CanvasHeight:   640,
		Columns:        3,
		Gap:            20,
		Padding:        20,
		BorderWidth:    1,
		Indent:         20,
		Outdent:        20,
		ProgressHeight: 16,

		ViewportWidth: 375,
		TimeoutMs:     30000,
		NetworkConditions: ports.NetworkConditions{
			LatencyMs:     20,
			DownloadSpeed: 10 * 1024 * 1024 / 8,
			UploadSpeed:   5 * 1024 * 1024 / 8,
		},
		CPUThrottling: 4.0,

		BannerEnabled: false,
		BannerHeight:  80,

		ShowProgress: true,

		VideoCRF: 25,
		Bitrate:  2000,
		OutroMs:  2000,
		FPS:      30.0,
	}
}

// Orchestrator coordinates the execution of all pipeline stages.
type Orchestrator struct {
	layoutStage    pipeline.Stage[pipeline.LayoutInput, pipeline.LayoutResult]
	recordStage    pipeline.Stage[pipeline.RecordInput, pipeline.RecordResult]
	bannerStage    pipeline.Stage[pipeline.BannerInput, pipeline.BannerResult]
	compositeStage pipeline.Stage[pipeline.CompositeInput, pipeline.CompositeResult]
	encodeStage    pipeline.Stage[pipeline.EncodeInput, pipeline.EncodeResult]
	fs             ports.FileSystem
	sink           ports.DebugSink
	logger         ports.Logger
}

// New creates a new Orchestrator.
func New(
	layoutStage pipeline.Stage[pipeline.LayoutInput, pipeline.LayoutResult],
	recordStage pipeline.Stage[pipeline.RecordInput, pipeline.RecordResult],
	bannerStage pipeline.Stage[pipeline.BannerInput, pipeline.BannerResult],
	compositeStage pipeline.Stage[pipeline.CompositeInput, pipeline.CompositeResult],
	encodeStage pipeline.Stage[pipeline.EncodeInput, pipeline.EncodeResult],
	fs ports.FileSystem,
	sink ports.DebugSink,
	logger ports.Logger,
) *Orchestrator {
	return &Orchestrator{
		layoutStage:    layoutStage,
		recordStage:    recordStage,
		bannerStage:    bannerStage,
		compositeStage: compositeStage,
		encodeStage:    encodeStage,
		fs:             fs,
		sink:           sink,
		logger:         logger,
	}
}

// Run executes the complete pipeline.
func (o *Orchestrator) Run(ctx context.Context, config Config) (RunResult, error) {
	o.logger.Info(l10n.T("Starting pipeline"))

	// 1. Layout calculation
	o.logger.Info(l10n.T("Calculating layout"))
	layoutInput := o.buildLayoutInput(config)
	layout, err := o.layoutStage.Execute(ctx, layoutInput)
	if err != nil {
		o.logger.Error(l10n.F("Failed to calculate layout: %s", err))
		return RunResult{}, fmt.Errorf("layout stage: %w", err)
	}
	o.logger.Info(l10n.F("Layout calculated: %dx%d canvas, %d columns", config.CanvasWidth, config.CanvasHeight, config.Columns))

	// Save layout debug output
	if o.sink.Enabled() {
		if data, err := json.MarshalIndent(layout, "", "  "); err == nil {
			o.sink.SaveLayoutJSON(data)
		}
	}

	// 2. Record page
	recordInput := o.buildRecordInput(config, layout)
	record, err := o.recordStage.Execute(ctx, recordInput)
	if err != nil {
		o.logger.Error(l10n.F("Failed to record page: %s", err))
		return RunResult{}, fmt.Errorf("record stage: %w", err)
	}
	o.logger.Info(l10n.F("Recording completed in %d ms", record.Timing.TotalDurationMs))

	// Save recording debug output
	if o.sink.Enabled() {
		if data, err := json.MarshalIndent(record.Timing, "", "  "); err == nil {
			o.sink.SaveRecordingJSON(data)
		}
	}

	// 3. Generate banner (optional)
	var banner *pipeline.BannerResult
	if config.BannerEnabled {
		o.logger.Info(l10n.T("Generating banner"))
		bannerInput := o.buildBannerInput(config, record)
		b, err := o.bannerStage.Execute(ctx, bannerInput)
		if err != nil {
			o.logger.Error(l10n.F("Failed to generate banner: %s", err))
			return RunResult{}, fmt.Errorf("banner stage: %w", err)
		}
		banner = &b
	}

	// 4. Compose frames
	o.logger.Info(l10n.F("Compositing %d frames", len(record.Frames)))
	compositeInput := o.buildCompositeInput(config, layout, record, banner)
	composite, err := o.compositeStage.Execute(ctx, compositeInput)
	if err != nil {
		o.logger.Error(l10n.F("Failed to composite frames: %s", err))
		return RunResult{}, fmt.Errorf("composite stage: %w", err)
	}
	o.logger.Info(l10n.T("Composition completed"))

	// 5. Encode video
	o.logger.Info(l10n.F("Encoding video with CRF %d", config.VideoCRF))
	encodeInput := o.buildEncodeInput(config, composite)
	encoded, err := o.encodeStage.Execute(ctx, encodeInput)
	if err != nil {
		o.logger.Error(l10n.F("Failed to encode video: %s", err))
		return RunResult{}, fmt.Errorf("encode stage: %w", err)
	}
	o.logger.Info(l10n.F("Video encoded: %d bytes", len(encoded.VideoData)))

	// 6. Write output file
	if err := o.fs.WriteFile(config.OutputPath, encoded.VideoData); err != nil {
		o.logger.Error(l10n.F("Failed to write output: %s", err))
		return RunResult{}, fmt.Errorf("write output: %w", err)
	}

	o.logger.Info(l10n.T("Pipeline completed successfully"))

	// Build result for summary
	result := RunResult{
		DOMContentLoadedMs: record.Timing.DOMContentLoadedMs,
		LoadCompleteMs:     record.Timing.LoadCompleteMs,
		TotalDurationMs:    record.Timing.TotalDurationMs,
		TimedOut:           record.Timing.TimedOut,
		TimeoutSec:         record.Timing.TimeoutSec,
		TotalBytes:         getTotalBytes(record.Frames),
		PageTitle:          record.PageInfo.Title,
		PageURL:            record.PageInfo.URL,
		FrameCount:         len(record.Frames),
		VideoDuration:      encoded.DurationMs,
		VideoFileSize:      encoded.FileSize,
		CanvasWidth:        config.CanvasWidth,
		CanvasHeight:       config.CanvasHeight,
	}

	return result, nil
}

func (o *Orchestrator) buildLayoutInput(config Config) pipeline.LayoutInput {
	return pipeline.LayoutInput{
		CanvasWidth:    config.CanvasWidth,
		CanvasHeight:   config.CanvasHeight,
		Columns:        config.Columns,
		Gap:            config.Gap,
		Padding:        config.Padding,
		BorderWidth:    config.BorderWidth,
		Indent:         config.Indent,
		Outdent:        config.Outdent,
		BannerHeight:   conditionalInt(config.BannerEnabled, config.BannerHeight, 0),
		ProgressHeight: conditionalInt(config.ShowProgress, config.ProgressHeight, 0),
	}
}

func (o *Orchestrator) buildRecordInput(config Config, layout pipeline.LayoutResult) pipeline.RecordInput {
	return pipeline.RecordInput{
		URL:               config.URL,
		ViewportWidth:     config.ViewportWidth, // Browser viewport width (e.g., 375 for mobile)
		Screen:            layout.Scroll,        // Target screen dimensions from layout
		ScreencastQuality: config.ScreencastQuality,
		TimeoutMs:         config.TimeoutMs,
		NetworkConditions: config.NetworkConditions,
		CPUThrottling:     config.CPUThrottling,
		Headers:           config.Headers,
		IgnoreHTTPSErrors: config.IgnoreHTTPSErrors,
		ProxyServer:       config.ProxyServer,
	}
}

func (o *Orchestrator) buildBannerInput(config Config, record pipeline.RecordResult) pipeline.BannerInput {
	return pipeline.BannerInput{
		Width:      config.CanvasWidth,
		Height:     config.BannerHeight,
		URL:        config.URL,
		Title:      record.PageInfo.Title,
		LoadTimeMs: record.Timing.LoadCompleteMs,
		TotalBytes: getTotalBytes(record.Frames),
		Credit:     config.Credit,
		Theme:      pipeline.DefaultBannerTheme(),
		TimedOut:   record.Timing.TimedOut,
		TimeoutSec: record.Timing.TimeoutSec,
	}
}

func (o *Orchestrator) buildCompositeInput(
	config Config,
	layout pipeline.LayoutResult,
	record pipeline.RecordResult,
	banner *pipeline.BannerResult,
) pipeline.CompositeInput {
	theme := pipeline.DefaultCompositeTheme()
	// Override theme colors if specified
	if config.BackgroundColor != [4]uint8{} {
		theme.BackgroundColor = rgbaFromArray(config.BackgroundColor)
	}
	if config.BorderColor != [4]uint8{} {
		theme.BorderColor = rgbaFromArray(config.BorderColor)
	}

	return pipeline.CompositeInput{
		RawFrames:    record.Frames,
		Layout:       layout,
		Banner:       banner,
		Theme:        theme,
		ShowProgress: config.ShowProgress,
		TotalTimeMs:  record.Timing.TotalDurationMs,
		TotalBytes:   getTotalBytes(record.Frames),
	}
}

func (o *Orchestrator) buildEncodeInput(config Config, composite pipeline.CompositeResult) pipeline.EncodeInput {
	return pipeline.EncodeInput{
		Frames:   composite.Frames,
		OutroMs:  config.OutroMs,
		VideoCRF: config.VideoCRF,
		Bitrate:  config.Bitrate,
		FPS:      config.FPS,
	}
}

func conditionalInt(condition bool, trueVal, falseVal int) int {
	if condition {
		return trueVal
	}
	return falseVal
}

func getTotalBytes(frames []pipeline.RawFrame) int64 {
	if len(frames) == 0 {
		return 0
	}
	return frames[len(frames)-1].TotalBytes
}

func rgbaFromArray(c [4]uint8) color.RGBA {
	return color.RGBA{R: c[0], G: c[1], B: c[2], A: c[3]}
}

// RunResult contains the results of a pipeline run for summary generation.
type RunResult struct {
	// Timing information
	DOMContentLoadedMs int
	LoadCompleteMs     int
	TotalDurationMs    int
	TimedOut           bool // True if recording ended due to timeout
	TimeoutSec         int  // Timeout value in seconds

	// Traffic information
	TotalBytes int64

	// Page information
	PageTitle string
	PageURL   string

	// Video information
	FrameCount    int
	VideoDuration int // in ms (includes outro)
	VideoFileSize int64

	// Layout information
	CanvasWidth  int
	CanvasHeight int
}
