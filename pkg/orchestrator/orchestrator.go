// Package orchestrator coordinates all pipeline stages.
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"

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

	// Recording
	ViewportWidth     int
	TimeoutMs         int
	NetworkConditions ports.NetworkConditions
	CPUThrottling     float64
	Headers           map[string]string

	// Banner
	BannerEnabled bool
	BannerHeight  int

	// Composition
	ShowProgress bool

	// Encoding
	Quality int
	Bitrate int
	OutroMs int
	FPS     float64
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

		Quality: 20,
		Bitrate: 2000,
		OutroMs: 2000,
		FPS:     30.0,
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
) *Orchestrator {
	return &Orchestrator{
		layoutStage:    layoutStage,
		recordStage:    recordStage,
		bannerStage:    bannerStage,
		compositeStage: compositeStage,
		encodeStage:    encodeStage,
		fs:             fs,
		sink:           sink,
	}
}

// Run executes the complete pipeline.
func (o *Orchestrator) Run(ctx context.Context, config Config) error {
	// 1. Layout calculation
	layoutInput := o.buildLayoutInput(config)
	layout, err := o.layoutStage.Execute(ctx, layoutInput)
	if err != nil {
		return fmt.Errorf("layout stage: %w", err)
	}

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
		return fmt.Errorf("record stage: %w", err)
	}

	// Save recording debug output
	if o.sink.Enabled() {
		if data, err := json.MarshalIndent(record.Timing, "", "  "); err == nil {
			o.sink.SaveRecordingJSON(data)
		}
	}

	// 3. Generate banner (optional)
	var banner *pipeline.BannerResult
	if config.BannerEnabled {
		bannerInput := o.buildBannerInput(config, record)
		b, err := o.bannerStage.Execute(ctx, bannerInput)
		if err != nil {
			return fmt.Errorf("banner stage: %w", err)
		}
		banner = &b
	}

	// 4. Compose frames
	compositeInput := o.buildCompositeInput(config, layout, record, banner)
	composite, err := o.compositeStage.Execute(ctx, compositeInput)
	if err != nil {
		return fmt.Errorf("composite stage: %w", err)
	}

	// 5. Encode video
	encodeInput := o.buildEncodeInput(config, composite)
	encoded, err := o.encodeStage.Execute(ctx, encodeInput)
	if err != nil {
		return fmt.Errorf("encode stage: %w", err)
	}

	// 6. Write output file
	if err := o.fs.WriteFile(config.OutputPath, encoded.VideoData); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
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
		ViewportWidth:     config.ViewportWidth,      // Browser viewport width (e.g., 375 for mobile)
		Screen:            layout.Scroll,             // Target screen dimensions from layout
		TimeoutMs:         config.TimeoutMs,
		NetworkConditions: config.NetworkConditions,
		CPUThrottling:     config.CPUThrottling,
		Headers:           config.Headers,
	}
}

func (o *Orchestrator) buildBannerInput(config Config, record pipeline.RecordResult) pipeline.BannerInput {
	return pipeline.BannerInput{
		Width:      config.CanvasWidth,
		Height:     config.BannerHeight,
		URL:        config.URL,
		Title:      record.PageInfo.Title,
		LoadTimeMs: record.Timing.TotalDurationMs,
		TotalBytes: getTotalBytes(record.Frames),
		Theme:      pipeline.DefaultBannerTheme(),
	}
}

func (o *Orchestrator) buildCompositeInput(
	config Config,
	layout pipeline.LayoutResult,
	record pipeline.RecordResult,
	banner *pipeline.BannerResult,
) pipeline.CompositeInput {
	return pipeline.CompositeInput{
		RawFrames:    record.Frames,
		Layout:       layout,
		Banner:       banner,
		Theme:        pipeline.DefaultCompositeTheme(),
		ShowProgress: config.ShowProgress,
		TotalTimeMs:  record.Timing.TotalDurationMs,
	}
}

func (o *Orchestrator) buildEncodeInput(config Config, composite pipeline.CompositeResult) pipeline.EncodeInput {
	return pipeline.EncodeInput{
		Frames:  composite.Frames,
		OutroMs: config.OutroMs,
		Quality: config.Quality,
		Bitrate: config.Bitrate,
		FPS:     config.FPS,
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
