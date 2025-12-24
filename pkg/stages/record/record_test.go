package record

import (
	"context"
	"testing"
	"time"

	"github.com/user/loadshow/pkg/mocks"
	"github.com/user/loadshow/pkg/pipeline"
	"github.com/user/loadshow/pkg/ports"
)

func TestStage_Execute(t *testing.T) {
	// Create mock browser that sends frames
	mockBrowser := &mocks.Browser{
		StartScreencastFunc: func(quality, maxWidth, maxHeight int) (<-chan ports.ScreenFrame, error) {
			ch := make(chan ports.ScreenFrame)
			go func() {
				defer close(ch)
				// Send a few frames
				for i := 0; i < 3; i++ {
					ch <- ports.ScreenFrame{
						TimestampMs: i * 100,
						Data:        []byte{0xFF, 0xD8, 0xFF}, // Fake JPEG header
						Metadata: ports.ScreenFrameMetadata{
							LoadedResources: i + 1,
							TotalResources:  5,
							TotalBytes:      int64((i + 1) * 1000),
						},
					}
					time.Sleep(10 * time.Millisecond)
				}
			}()
			return ch, nil
		},
		GetPageInfoFunc: func() (*ports.PageInfo, error) {
			return &ports.PageInfo{
				Title:        "Test Page",
				URL:          "https://example.com",
				ScrollHeight: 1000,
				ScrollWidth:  375,
			}, nil
		},
	}

	mockSink := mocks.NewDebugSink(false)

	stage := New(mockBrowser, mockSink, ports.BrowserOptions{Headless: true})

	input := pipeline.DefaultRecordInput()
	input.URL = "https://example.com"
	input.Screen = pipeline.Dimension{Width: 144, Height: 1739} // 3-column layout scroll dimensions
	input.TimeoutMs = 5000

	result, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check frames were collected
	if len(result.Frames) != 3 {
		t.Errorf("expected 3 frames, got %d", len(result.Frames))
	}

	// Check frame timestamps
	expectedTimestamps := []int{0, 100, 200}
	for i, frame := range result.Frames {
		if frame.TimestampMs != expectedTimestamps[i] {
			t.Errorf("frame %d: expected timestamp %d, got %d",
				i, expectedTimestamps[i], frame.TimestampMs)
		}
	}

	// Check page info
	if result.PageInfo.Title != "Test Page" {
		t.Errorf("expected title 'Test Page', got '%s'", result.PageInfo.Title)
	}
}

func TestStage_Execute_WithDebugSink(t *testing.T) {
	mockBrowser := &mocks.Browser{
		StartScreencastFunc: func(quality, maxWidth, maxHeight int) (<-chan ports.ScreenFrame, error) {
			ch := make(chan ports.ScreenFrame)
			go func() {
				defer close(ch)
				ch <- ports.ScreenFrame{
					TimestampMs: 0,
					Data:        []byte{0xFF, 0xD8, 0xFF},
				}
			}()
			return ch, nil
		},
		GetPageInfoFunc: func() (*ports.PageInfo, error) {
			return &ports.PageInfo{}, nil
		},
	}

	// Enable debug sink
	mockSink := mocks.NewDebugSink(true)

	stage := New(mockBrowser, mockSink, ports.BrowserOptions{Headless: true})

	input := pipeline.DefaultRecordInput()
	input.URL = "https://example.com"
	input.Screen = pipeline.Dimension{Width: 144, Height: 1739}
	input.TimeoutMs = 1000

	_, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that frame was saved to debug sink
	if len(mockSink.RawFrames) != 1 {
		t.Errorf("expected 1 raw frame in sink, got %d", len(mockSink.RawFrames))
	}
}

func TestStage_Execute_Timeout(t *testing.T) {
	// Create a browser that sends frames slowly
	mockBrowser := &mocks.Browser{
		StartScreencastFunc: func(quality, maxWidth, maxHeight int) (<-chan ports.ScreenFrame, error) {
			ch := make(chan ports.ScreenFrame)
			go func() {
				defer close(ch)
				for i := 0; i < 100; i++ {
					ch <- ports.ScreenFrame{
						TimestampMs: i * 100,
						Data:        []byte{0xFF},
					}
					time.Sleep(50 * time.Millisecond)
				}
			}()
			return ch, nil
		},
		GetPageInfoFunc: func() (*ports.PageInfo, error) {
			return &ports.PageInfo{}, nil
		},
	}

	mockSink := mocks.NewDebugSink(false)
	stage := New(mockBrowser, mockSink, ports.BrowserOptions{Headless: true})

	input := pipeline.DefaultRecordInput()
	input.URL = "https://example.com"
	input.Screen = pipeline.Dimension{Width: 144, Height: 1739}
	input.TimeoutMs = 200 // Short timeout

	result, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have collected some frames before timeout
	if len(result.Frames) == 0 {
		t.Error("expected some frames before timeout")
	}
	if len(result.Frames) >= 100 {
		t.Error("expected timeout to stop frame collection")
	}
}
