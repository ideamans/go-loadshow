// Package integration contains integration tests for the loadshow pipeline.
package integration

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"os"
	"testing"
	"time"

	"github.com/user/loadshow/pkg/adapters/av1decoder"
	"github.com/user/loadshow/pkg/adapters/av1encoder"
	"github.com/user/loadshow/pkg/adapters/capturehtml"
	"github.com/user/loadshow/pkg/adapters/chromebrowser"
	"github.com/user/loadshow/pkg/adapters/filesink"
	"github.com/user/loadshow/pkg/adapters/ggrenderer"
	"github.com/user/loadshow/pkg/adapters/h264decoder"
	"github.com/user/loadshow/pkg/adapters/h264encoder"
	"github.com/user/loadshow/pkg/adapters/logger"
	"github.com/user/loadshow/pkg/adapters/nullsink"
	"github.com/user/loadshow/pkg/adapters/osfilesystem"
	"github.com/user/loadshow/pkg/orchestrator"
	"github.com/user/loadshow/pkg/pipeline"
	"github.com/user/loadshow/pkg/ports"
	"github.com/user/loadshow/pkg/stages/banner"
	"github.com/user/loadshow/pkg/stages/composite"
	"github.com/user/loadshow/pkg/stages/encode"
	"github.com/user/loadshow/pkg/stages/layout"
	"github.com/user/loadshow/pkg/stages/record"
)

// Suppress unused import warnings for packages that may only be used in skipped tests
var (
	_ = capturehtml.New
	_ = chromebrowser.New
	_ = filesink.New
	_ = osfilesystem.New
	_ = record.New
	_ = banner.NewStage
)

// TestLayoutToComposite tests the layout → composite pipeline
func TestLayoutToComposite(t *testing.T) {
	// Create layout stage
	layoutStage := layout.NewStage()

	layoutInput := pipeline.LayoutInput{
		CanvasWidth:    512,
		CanvasHeight:   640,
		Columns:        3,
		Gap:            20,
		Padding:        20,
		BorderWidth:    1,
		Indent:         20,
		Outdent:        20,
		BannerHeight:   80,
		ProgressHeight: 16,
	}

	layoutResult, err := layoutStage.Execute(context.Background(), layoutInput)
	if err != nil {
		t.Fatalf("Layout stage failed: %v", err)
	}

	// Verify layout result
	if len(layoutResult.Windows) != 3 {
		t.Errorf("expected 3 windows, got %d", len(layoutResult.Windows))
	}

	// Create fake raw frames for composite
	rawFrames := createFakeRawFrames(10, layoutResult.Scroll.Width, layoutResult.Scroll.Height)

	// Create composite stage
	renderer := ggrenderer.New()
	compositeStage := composite.NewStage(renderer, nullsink.New(), logger.NewNoop(), 2)

	compositeInput := pipeline.CompositeInput{
		RawFrames:    rawFrames,
		Layout:       layoutResult,
		Banner:       nil,
		Theme:        pipeline.DefaultCompositeTheme(),
		ShowProgress: true,
		TotalTimeMs:  1000,
		TotalBytes:   1024 * 100,
	}

	compositeResult, err := compositeStage.Execute(context.Background(), compositeInput)
	if err != nil {
		t.Fatalf("Composite stage failed: %v", err)
	}

	// Verify composite result
	if len(compositeResult.Frames) != len(rawFrames) {
		t.Errorf("expected %d frames, got %d", len(rawFrames), len(compositeResult.Frames))
	}

	// Check frame dimensions - composite produces canvas-sized output
	if len(compositeResult.Frames) > 0 {
		bounds := compositeResult.Frames[0].Image.Bounds()
		if bounds.Dx() != layoutInput.CanvasWidth {
			t.Errorf("expected frame width %d, got %d", layoutInput.CanvasWidth, bounds.Dx())
		}
		// Height may include banner + progress bar area based on layout
		if bounds.Dy() < layoutInput.CanvasHeight {
			t.Errorf("frame height %d is less than canvas height %d", bounds.Dy(), layoutInput.CanvasHeight)
		}
	}
}

// TestCompositeToEncode tests the composite → encode pipeline
func TestCompositeToEncode(t *testing.T) {
	// Create composed frames
	composedFrames := make([]pipeline.ComposedFrame, 10)
	for i := 0; i < 10; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 256, 320))
		// Fill with gradient
		for y := 0; y < 320; y++ {
			for x := 0; x < 256; x++ {
				c := color.RGBA{
					R: uint8(x),
					G: uint8(y % 256),
					B: uint8((i * 25) % 256),
					A: 255,
				}
				img.Set(x, y, c)
			}
		}
		composedFrames[i] = pipeline.ComposedFrame{
			TimestampMs: i * 100,
			Image:       img,
		}
	}

	// Create encode stage
	encoder := av1encoder.New()
	encodeStage := encode.NewStage(encoder, logger.NewNoop())

	encodeInput := pipeline.EncodeInput{
		Frames:  composedFrames,
		OutroMs: 500,
		VideoCRF: 40,
		Bitrate: 1000,
		FPS:     30.0,
	}

	encodeResult, err := encodeStage.Execute(context.Background(), encodeInput)
	if err != nil {
		t.Fatalf("Encode stage failed: %v", err)
	}

	// Verify encode result
	if len(encodeResult.VideoData) == 0 {
		t.Error("expected non-empty video data")
	}

	// Verify it's a valid MP4
	if len(encodeResult.VideoData) < 8 {
		t.Fatal("video data too short")
	}
	if string(encodeResult.VideoData[4:8]) != "ftyp" {
		t.Error("expected ftyp signature in video data")
	}
}

// TestCompositeToEncodeH264 tests the composite → encode pipeline with H.264 codec
func TestCompositeToEncodeH264(t *testing.T) {
	if !h264encoder.IsAvailable() {
		t.Skip("H.264 encoder not available")
	}

	// Create composed frames
	composedFrames := make([]pipeline.ComposedFrame, 10)
	for i := 0; i < 10; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 256, 320))
		// Fill with gradient
		for y := 0; y < 320; y++ {
			for x := 0; x < 256; x++ {
				c := color.RGBA{
					R: uint8(x),
					G: uint8(y % 256),
					B: uint8((i * 25) % 256),
					A: 255,
				}
				img.Set(x, y, c)
			}
		}
		composedFrames[i] = pipeline.ComposedFrame{
			TimestampMs: i * 100,
			Image:       img,
		}
	}

	// Create encode stage with H.264
	encoder := h264encoder.New()
	encodeStage := encode.NewStage(encoder, logger.NewNoop())

	encodeInput := pipeline.EncodeInput{
		Frames:   composedFrames,
		OutroMs:  500,
		VideoCRF: 40,
		Bitrate:  1000,
		FPS:      30.0,
	}

	encodeResult, err := encodeStage.Execute(context.Background(), encodeInput)
	if err != nil {
		t.Fatalf("Encode stage failed: %v", err)
	}

	// Verify encode result
	if len(encodeResult.VideoData) == 0 {
		t.Error("expected non-empty video data")
	}

	// Verify it's a valid MP4
	if len(encodeResult.VideoData) < 8 {
		t.Fatal("video data too short")
	}
	if string(encodeResult.VideoData[4:8]) != "ftyp" {
		t.Error("expected ftyp signature in video data")
	}

	t.Logf("H.264 encoded video: %d bytes", len(encodeResult.VideoData))
}

// TestFullPipelineWithH264 tests the full pipeline with H.264 codec
func TestFullPipelineWithH264(t *testing.T) {
	if !h264encoder.IsAvailable() {
		t.Skip("H.264 encoder not available")
	}

	// Create all stages with real adapters except browser
	layoutStage := layout.NewStage()
	renderer := ggrenderer.New()
	encoder := h264encoder.New()

	// Layout
	layoutInput := pipeline.LayoutInput{
		CanvasWidth:    256,
		CanvasHeight:   320,
		Columns:        2,
		Gap:            10,
		Padding:        10,
		BorderWidth:    1,
		Indent:         10,
		Outdent:        10,
		BannerHeight:   0,
		ProgressHeight: 8,
	}

	layoutResult, err := layoutStage.Execute(context.Background(), layoutInput)
	if err != nil {
		t.Fatalf("Layout failed: %v", err)
	}

	// Create mock raw frames
	rawFrames := createFakeRawFrames(5, layoutResult.Scroll.Width, layoutResult.Scroll.Height)

	// Composite
	compositeStage := composite.NewStage(renderer, nullsink.New(), logger.NewNoop(), 2)
	compositeInput := pipeline.CompositeInput{
		RawFrames:    rawFrames,
		Layout:       layoutResult,
		Theme:        pipeline.DefaultCompositeTheme(),
		ShowProgress: true,
		TotalTimeMs:  500,
		TotalBytes:   50000,
	}

	compositeResult, err := compositeStage.Execute(context.Background(), compositeInput)
	if err != nil {
		t.Fatalf("Composite failed: %v", err)
	}

	// Encode
	encodeStage := encode.NewStage(encoder, logger.NewNoop())
	encodeInput := pipeline.EncodeInput{
		Frames:   compositeResult.Frames,
		OutroMs:  200,
		VideoCRF: 45,
		FPS:      30.0,
	}

	encodeResult, err := encodeStage.Execute(context.Background(), encodeInput)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Verify output
	if len(encodeResult.VideoData) == 0 {
		t.Error("expected video output")
	}

	// Decode with H.264 decoder and verify frame count
	frames, err := h264decoder.ExtractFrames(encodeResult.VideoData)
	if err != nil {
		t.Fatalf("ExtractFrames failed: %v", err)
	}

	if len(frames) < len(rawFrames) {
		t.Errorf("expected at least %d frames, got %d", len(rawFrames), len(frames))
	}

	t.Logf("H.264 full pipeline: %d bytes, %d frames", len(encodeResult.VideoData), len(frames))
}

// TestBannerStageWithRealCapturer tests the banner stage with real HTML capturer
func TestBannerStageWithRealCapturer(t *testing.T) {
	if os.Getenv("CI") != "" && os.Getenv("CHROME_PATH") == "" {
		t.Skip("Skipping browser test in CI without Chrome")
	}

	capturer := capturehtml.New()
	bannerStage := banner.NewStage(capturer, nullsink.New(), logger.NewNoop())

	input := pipeline.BannerInput{
		Width:      400,
		Height:     80,
		URL:        "https://example.com",
		Title:      "Integration Test",
		LoadTimeMs: 1500,
		TotalBytes: 1024 * 500,
		Credit:     "loadshow",
		Theme:      pipeline.DefaultBannerTheme(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := bannerStage.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Banner stage failed: %v", err)
	}

	if result.Image == nil {
		t.Error("expected banner image")
	}

	bounds := result.Image.Bounds()
	if bounds.Dx() != input.Width {
		t.Errorf("expected banner width %d, got %d", input.Width, bounds.Dx())
	}
}

// TestFullPipelineWithMockBrowser tests the full pipeline with mock browser data
func TestFullPipelineWithMockBrowser(t *testing.T) {
	// Create all stages with real adapters except browser
	layoutStage := layout.NewStage()
	renderer := ggrenderer.New()
	encoder := av1encoder.New()

	// Layout
	layoutInput := pipeline.LayoutInput{
		CanvasWidth:    256,
		CanvasHeight:   320,
		Columns:        2,
		Gap:            10,
		Padding:        10,
		BorderWidth:    1,
		Indent:         10,
		Outdent:        10,
		BannerHeight:   0,
		ProgressHeight: 8,
	}

	layoutResult, err := layoutStage.Execute(context.Background(), layoutInput)
	if err != nil {
		t.Fatalf("Layout failed: %v", err)
	}

	// Create mock raw frames
	rawFrames := createFakeRawFrames(5, layoutResult.Scroll.Width, layoutResult.Scroll.Height)

	// Composite
	compositeStage := composite.NewStage(renderer, nullsink.New(), logger.NewNoop(), 2)
	compositeInput := pipeline.CompositeInput{
		RawFrames:    rawFrames,
		Layout:       layoutResult,
		Theme:        pipeline.DefaultCompositeTheme(),
		ShowProgress: true,
		TotalTimeMs:  500,
		TotalBytes:   50000,
	}

	compositeResult, err := compositeStage.Execute(context.Background(), compositeInput)
	if err != nil {
		t.Fatalf("Composite failed: %v", err)
	}

	// Encode
	encodeStage := encode.NewStage(encoder, logger.NewNoop())
	encodeInput := pipeline.EncodeInput{
		Frames:  compositeResult.Frames,
		OutroMs: 200,
		VideoCRF: 45,
		FPS:     30.0,
	}

	encodeResult, err := encodeStage.Execute(context.Background(), encodeInput)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Verify output
	if len(encodeResult.VideoData) == 0 {
		t.Error("expected video output")
	}

	// Decode and verify frame count
	frames, err := av1decoder.ExtractFrames(encodeResult.VideoData)
	if err != nil {
		t.Fatalf("ExtractFrames failed: %v", err)
	}

	if len(frames) < len(rawFrames) {
		t.Errorf("expected at least %d frames, got %d", len(rawFrames), len(frames))
	}
}

// TestRecordStageWithRealBrowser tests recording with real browser
func TestRecordStageWithRealBrowser(t *testing.T) {
	if os.Getenv("CI") != "" && os.Getenv("CHROME_PATH") == "" {
		t.Skip("Skipping browser test in CI without Chrome")
	}

	browser := chromebrowser.New()
	recordStage := record.New(browser, nullsink.New(), logger.NewNoop(), ports.BrowserOptions{
		Headless:  true,
		Incognito: true,
	})

	input := pipeline.RecordInput{
		URL:           "data:text/html,<html><body><h1>Test</h1></body></html>",
		ViewportWidth: 375,
		Screen:        pipeline.Dimension{Width: 100, Height: 500},
		TimeoutMs:     5000,
		NetworkConditions: ports.NetworkConditions{
			LatencyMs:     10,
			DownloadSpeed: 10 * 1024 * 1024 / 8,
			UploadSpeed:   10 * 1024 * 1024 / 8,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := recordStage.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Record stage failed: %v", err)
	}

	if len(result.Frames) == 0 {
		t.Error("expected at least one frame")
	}

	if result.PageInfo.Title == "" {
		t.Log("Warning: page title is empty")
	}
}

// TestOrchestratorWithDebugSink tests orchestrator with debug output
func TestOrchestratorWithDebugSink(t *testing.T) {
	if os.Getenv("CI") != "" && os.Getenv("CHROME_PATH") == "" {
		t.Skip("Skipping browser test in CI without Chrome")
	}

	// Create temp directory for debug output
	tmpDir, err := os.MkdirTemp("", "loadshow-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create adapters
	fs := osfilesystem.New()
	renderer := ggrenderer.New()
	browser := chromebrowser.New()
	htmlCapturer := capturehtml.New()
	encoder := av1encoder.New()
	sink := filesink.New(tmpDir, fs, renderer)

	// Create stages
	layoutStage := layout.NewStage()
	recordStage := record.New(browser, sink, logger.NewNoop(), ports.BrowserOptions{Headless: true, Incognito: true})
	bannerStage := banner.NewStage(htmlCapturer, sink, logger.NewNoop())
	compositeStage := composite.NewStage(renderer, sink, logger.NewNoop(), 2)
	encodeStage := encode.NewStage(encoder, logger.NewNoop())

	// Create orchestrator
	orch := orchestrator.New(
		layoutStage,
		recordStage,
		bannerStage,
		compositeStage,
		encodeStage,
		fs,
		sink,
		logger.NewNoop(),
	)

	// Create config
	config := orchestrator.Config{
		URL:           "data:text/html,<html><body><h1>Debug Test</h1><p>Content</p></body></html>",
		OutputPath:    tmpDir + "/output.mp4",
		CanvasWidth:   256,
		CanvasHeight:  320,
		Columns:       2,
		Gap:           10,
		Padding:       10,
		BorderWidth:   1,
		ViewportWidth: 375,
		TimeoutMs:     10000,
		BannerEnabled: true,
		BannerHeight:  40,
		ShowProgress:  true,
		VideoCRF:       45,
		OutroMs:       200,
		FPS:           30.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = orch.Run(ctx, config)
	if err != nil {
		t.Fatalf("Orchestrator failed: %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(config.OutputPath); os.IsNotExist(err) {
		t.Error("expected output file to exist")
	}

	// Verify debug files exist
	if _, err := os.Stat(tmpDir + "/layout.json"); os.IsNotExist(err) {
		t.Error("expected layout.json in debug output")
	}
}

// createFakeRawFrames creates fake raw frames for testing
func createFakeRawFrames(count, width, height int) []pipeline.RawFrame {
	frames := make([]pipeline.RawFrame, count)

	for i := 0; i < count; i++ {
		// Create a simple JPEG-like image data
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				c := color.RGBA{
					R: uint8((x + i*20) % 256),
					G: uint8((y + i*10) % 256),
					B: uint8(i * 25 % 256),
					A: 255,
				}
				img.Set(x, y, c)
			}
		}

		// Encode as JPEG
		var buf bytes.Buffer
		renderer := ggrenderer.New()
		jpegData, _ := renderer.EncodeImage(img, ports.FormatJPEG, 80)

		frames[i] = pipeline.RawFrame{
			TimestampMs:     i * 100,
			ImageData:       jpegData,
			LoadedResources: i + 1,
			TotalResources:  count,
			TotalBytes:      int64((i + 1) * 10000),
		}
		_ = buf // prevent unused variable warning
	}

	return frames
}
