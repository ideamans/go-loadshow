package encode

import (
	"context"
	"image"
	"testing"

	"github.com/user/loadshow/pkg/adapters/logger"
	"github.com/user/loadshow/pkg/mocks"
	"github.com/user/loadshow/pkg/pipeline"
)

func TestStage_Execute(t *testing.T) {
	mockEncoder := &mocks.VideoEncoder{}

	stage := NewStage(mockEncoder, logger.NewNoop())

	// Create test frames
	frames := []pipeline.ComposedFrame{
		{TimestampMs: 0, Image: image.NewRGBA(image.Rect(0, 0, 512, 640))},
		{TimestampMs: 100, Image: image.NewRGBA(image.Rect(0, 0, 512, 640))},
		{TimestampMs: 200, Image: image.NewRGBA(image.Rect(0, 0, 512, 640))},
	}

	input := pipeline.EncodeInput{
		Frames:   frames,

		VideoCRF: 30,
		Bitrate:  1000,
		FPS:      30.0,
	}

	result, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check encoder was called correctly
	if !mockEncoder.BeginCalled {
		t.Error("expected Begin to be called")
	}
	if !mockEncoder.EndCalled {
		t.Error("expected End to be called")
	}

	// Check frame count (3 frames, no outro frame)
	expectedFrameCalls := 3
	if len(mockEncoder.EncodeFrameCalls) != expectedFrameCalls {
		t.Errorf("expected %d EncodeFrame calls, got %d",
			expectedFrameCalls, len(mockEncoder.EncodeFrameCalls))
	}

	// Check duration is last frame timestamp
	expectedDuration := 200
	if result.DurationMs != expectedDuration {
		t.Errorf("expected duration %d, got %d", expectedDuration, result.DurationMs)
	}

	// Check video data is returned
	if len(result.VideoData) == 0 {
		t.Error("expected video data to be returned")
	}
}

func TestStage_Execute_NoOutro(t *testing.T) {
	mockEncoder := &mocks.VideoEncoder{}

	stage := NewStage(mockEncoder, logger.NewNoop())

	frames := []pipeline.ComposedFrame{
		{TimestampMs: 0, Image: image.NewRGBA(image.Rect(0, 0, 512, 640))},
		{TimestampMs: 100, Image: image.NewRGBA(image.Rect(0, 0, 512, 640))},
	}

	input := pipeline.EncodeInput{
		Frames:   frames,

		VideoCRF: 30,
		Bitrate:  1000,
		FPS:      30.0,
	}

	_, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check frame count (2 frames, no outro)
	if len(mockEncoder.EncodeFrameCalls) != 2 {
		t.Errorf("expected 2 EncodeFrame calls, got %d", len(mockEncoder.EncodeFrameCalls))
	}
}

func TestStage_Execute_EmptyFrames(t *testing.T) {
	mockEncoder := &mocks.VideoEncoder{}

	stage := NewStage(mockEncoder, logger.NewNoop())

	input := pipeline.EncodeInput{
		Frames: []pipeline.ComposedFrame{},
	}

	_, err := stage.Execute(context.Background(), input)
	if err == nil {
		t.Error("expected error for empty frames")
	}
}

func TestStage_Execute_ContextCancelled(t *testing.T) {
	mockEncoder := &mocks.VideoEncoder{}

	stage := NewStage(mockEncoder, logger.NewNoop())

	frames := []pipeline.ComposedFrame{
		{TimestampMs: 0, Image: image.NewRGBA(image.Rect(0, 0, 512, 640))},
		{TimestampMs: 100, Image: image.NewRGBA(image.Rect(0, 0, 512, 640))},
	}

	input := pipeline.EncodeInput{
		Frames: frames,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := stage.Execute(ctx, input)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestStage_Execute_FrameTimestamps(t *testing.T) {
	mockEncoder := &mocks.VideoEncoder{}

	stage := NewStage(mockEncoder, logger.NewNoop())

	frames := []pipeline.ComposedFrame{
		{TimestampMs: 0, Image: image.NewRGBA(image.Rect(0, 0, 100, 100))},
		{TimestampMs: 500, Image: image.NewRGBA(image.Rect(0, 0, 100, 100))},
		{TimestampMs: 1000, Image: image.NewRGBA(image.Rect(0, 0, 100, 100))},
	}

	input := pipeline.EncodeInput{
		Frames:  frames,

	}

	_, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check timestamps (no outro frame added)
	expectedTimestamps := []int{0, 500, 1000}
	for i, call := range mockEncoder.EncodeFrameCalls {
		if call.TimestampMs != expectedTimestamps[i] {
			t.Errorf("call %d: expected timestamp %d, got %d",
				i, expectedTimestamps[i], call.TimestampMs)
		}
	}
}
