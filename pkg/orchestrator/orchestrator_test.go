package orchestrator

import (
	"context"
	"image"
	"testing"

	"github.com/user/loadshow/pkg/mocks"
	"github.com/user/loadshow/pkg/pipeline"
	"github.com/user/loadshow/pkg/ports"
)

// mockLayoutStage is a mock for the layout stage.
type mockLayoutStage struct {
	result pipeline.LayoutResult
	err    error
}

func (m *mockLayoutStage) Execute(ctx context.Context, input pipeline.LayoutInput) (pipeline.LayoutResult, error) {
	if m.err != nil {
		return pipeline.LayoutResult{}, m.err
	}
	return m.result, nil
}

// mockRecordStage is a mock for the record stage.
type mockRecordStage struct {
	result pipeline.RecordResult
	err    error
}

func (m *mockRecordStage) Execute(ctx context.Context, input pipeline.RecordInput) (pipeline.RecordResult, error) {
	if m.err != nil {
		return pipeline.RecordResult{}, m.err
	}
	return m.result, nil
}

// mockBannerStage is a mock for the banner stage.
type mockBannerStage struct {
	result pipeline.BannerResult
	err    error
}

func (m *mockBannerStage) Execute(ctx context.Context, input pipeline.BannerInput) (pipeline.BannerResult, error) {
	if m.err != nil {
		return pipeline.BannerResult{}, m.err
	}
	return m.result, nil
}

// mockCompositeStage is a mock for the composite stage.
type mockCompositeStage struct {
	result pipeline.CompositeResult
	err    error
}

func (m *mockCompositeStage) Execute(ctx context.Context, input pipeline.CompositeInput) (pipeline.CompositeResult, error) {
	if m.err != nil {
		return pipeline.CompositeResult{}, m.err
	}
	return m.result, nil
}

// mockEncodeStage is a mock for the encode stage.
type mockEncodeStage struct {
	result pipeline.EncodeResult
	err    error
}

func (m *mockEncodeStage) Execute(ctx context.Context, input pipeline.EncodeInput) (pipeline.EncodeResult, error) {
	if m.err != nil {
		return pipeline.EncodeResult{}, m.err
	}
	return m.result, nil
}

func TestOrchestrator_Run(t *testing.T) {
	// Create mock stages
	layoutStage := &mockLayoutStage{
		result: pipeline.LayoutResult{
			Scroll: pipeline.Dimension{Width: 142, Height: 1740},
			Columns: []pipeline.Rectangle{
				{X: 20, Y: 20, Width: 144, Height: 580},
			},
			Windows: []pipeline.Window{
				{Rectangle: pipeline.Rectangle{X: 21, Y: 21, Width: 142, Height: 578}, ScrollTop: 0},
			},
			ContentArea:  pipeline.Rectangle{X: 20, Y: 20, Width: 472, Height: 584},
			ProgressArea: pipeline.Rectangle{X: 20, Y: 604, Width: 472, Height: 16},
		},
	}

	recordStage := &mockRecordStage{
		result: pipeline.RecordResult{
			Frames: []pipeline.RawFrame{
				{TimestampMs: 0, ImageData: []byte{0xFF, 0xD8}},
				{TimestampMs: 100, ImageData: []byte{0xFF, 0xD8}},
			},
			PageInfo: ports.PageInfo{
				Title: "Test Page",
				URL:   "https://example.com",
			},
			Timing: pipeline.TimingInfo{
				TotalDurationMs: 100,
			},
		},
	}

	bannerStage := &mockBannerStage{
		result: pipeline.BannerResult{
			Image: image.NewRGBA(image.Rect(0, 0, 472, 80)),
		},
	}

	compositeStage := &mockCompositeStage{
		result: pipeline.CompositeResult{
			Frames: []pipeline.ComposedFrame{
				{TimestampMs: 0, Image: image.NewRGBA(image.Rect(0, 0, 512, 640))},
				{TimestampMs: 100, Image: image.NewRGBA(image.Rect(0, 0, 512, 640))},
			},
		},
	}

	encodeStage := &mockEncodeStage{
		result: pipeline.EncodeResult{
			VideoData:  []byte{0x00, 0x00, 0x00, 0x20}, // MP4 bytes
			DurationMs: 1100,
			FileSize:   4,
		},
	}

	mockFS := mocks.NewFileSystem()
	mockSink := mocks.NewDebugSink(false)

	orch := New(
		layoutStage,
		recordStage,
		bannerStage,
		compositeStage,
		encodeStage,
		mockFS,
		mockSink,
	)

	config := DefaultConfig()
	config.URL = "https://example.com"
	config.OutputPath = "output.webm"

	err := orch.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that output file was written
	exists, _ := mockFS.Exists("output.webm")
	if !exists {
		t.Error("expected output file to be written")
	}

	// Check file contents
	data, ok := mockFS.GetFile("output.webm")
	if !ok {
		t.Error("expected file to exist")
	}
	if len(data) == 0 {
		t.Error("expected file to have content")
	}
}

func TestOrchestrator_Run_WithBanner(t *testing.T) {
	layoutStage := &mockLayoutStage{
		result: pipeline.LayoutResult{
			Scroll:       pipeline.Dimension{Width: 142, Height: 1740},
			BannerArea:   pipeline.Rectangle{X: 20, Y: 20, Width: 472, Height: 80},
			ContentArea:  pipeline.Rectangle{X: 20, Y: 120, Width: 472, Height: 484},
			ProgressArea: pipeline.Rectangle{X: 20, Y: 604, Width: 472, Height: 16},
		},
	}

	recordStage := &mockRecordStage{
		result: pipeline.RecordResult{
			Frames: []pipeline.RawFrame{
				{TimestampMs: 0, ImageData: []byte{0xFF}},
			},
			Timing: pipeline.TimingInfo{TotalDurationMs: 100},
		},
	}

	bannerCalled := false
	bannerStage := &mockBannerStage{
		result: pipeline.BannerResult{
			Image: image.NewRGBA(image.Rect(0, 0, 472, 80)),
		},
	}
	// Wrap to track calls
	wrappedBannerStage := pipeline.StageFunc[pipeline.BannerInput, pipeline.BannerResult](
		func(ctx context.Context, input pipeline.BannerInput) (pipeline.BannerResult, error) {
			bannerCalled = true
			return bannerStage.Execute(ctx, input)
		},
	)

	compositeStage := &mockCompositeStage{
		result: pipeline.CompositeResult{
			Frames: []pipeline.ComposedFrame{
				{TimestampMs: 0, Image: image.NewRGBA(image.Rect(0, 0, 512, 640))},
			},
		},
	}

	encodeStage := &mockEncodeStage{
		result: pipeline.EncodeResult{VideoData: []byte{0x00}},
	}

	mockFS := mocks.NewFileSystem()
	mockSink := mocks.NewDebugSink(false)

	orch := New(
		layoutStage,
		recordStage,
		wrappedBannerStage,
		compositeStage,
		encodeStage,
		mockFS,
		mockSink,
	)

	config := DefaultConfig()
	config.URL = "https://example.com"
	config.OutputPath = "output.webm"
	config.BannerEnabled = true

	err := orch.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bannerCalled {
		t.Error("expected banner stage to be called when BannerEnabled is true")
	}
}

func TestOrchestrator_Run_WithDebugSink(t *testing.T) {
	layoutStage := &mockLayoutStage{
		result: pipeline.LayoutResult{
			Scroll: pipeline.Dimension{Width: 142, Height: 1740},
		},
	}

	recordStage := &mockRecordStage{
		result: pipeline.RecordResult{
			Frames: []pipeline.RawFrame{{TimestampMs: 0, ImageData: []byte{0xFF}}},
			Timing: pipeline.TimingInfo{TotalDurationMs: 100},
		},
	}

	bannerStage := &mockBannerStage{}

	compositeStage := &mockCompositeStage{
		result: pipeline.CompositeResult{
			Frames: []pipeline.ComposedFrame{
				{TimestampMs: 0, Image: image.NewRGBA(image.Rect(0, 0, 512, 640))},
			},
		},
	}

	encodeStage := &mockEncodeStage{
		result: pipeline.EncodeResult{VideoData: []byte{0x00}},
	}

	mockFS := mocks.NewFileSystem()
	mockSink := mocks.NewDebugSink(true) // Enable debug

	orch := New(
		layoutStage,
		recordStage,
		bannerStage,
		compositeStage,
		encodeStage,
		mockFS,
		mockSink,
	)

	config := DefaultConfig()
	config.URL = "https://example.com"
	config.OutputPath = "output.webm"

	err := orch.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that debug output was saved
	if len(mockSink.LayoutJSON) == 0 {
		t.Error("expected layout JSON to be saved")
	}
	if len(mockSink.RecordingJSON) == 0 {
		t.Error("expected recording JSON to be saved")
	}
}
