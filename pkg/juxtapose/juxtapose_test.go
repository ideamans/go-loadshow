package juxtapose

import (
	"context"
	"image"
	"image/color"
	"io"
	"testing"

	"github.com/user/loadshow/pkg/ports"
)

// mockDecoder is a test implementation of VideoDecoder.
type mockDecoder struct {
	leftFrames  []ports.VideoFrame
	rightFrames []ports.VideoFrame
	pathIndex   int
}

func (m *mockDecoder) ReadFrames(path string) ([]ports.VideoFrame, error) {
	if m.pathIndex == 0 {
		m.pathIndex++
		return m.leftFrames, nil
	}
	return m.rightFrames, nil
}

func (m *mockDecoder) ReadFramesFromReader(reader io.ReadSeeker) ([]ports.VideoFrame, error) {
	return nil, nil
}

func (m *mockDecoder) Close() {}

// mockEncoder captures encoding parameters for verification.
type mockEncoder struct {
	width   int
	height  int
	fps     float64
	frames  []image.Image
	started bool
}

func (m *mockEncoder) Begin(width, height int, fps float64, opts ports.EncoderOptions) error {
	m.width = width
	m.height = height
	m.fps = fps
	m.started = true
	return nil
}

func (m *mockEncoder) EncodeFrame(img image.Image, timestampMs int) error {
	m.frames = append(m.frames, img)
	return nil
}

func (m *mockEncoder) End() ([]byte, error) {
	return []byte{0x00}, nil
}

// mockFileSystem for testing.
type mockFileSystem struct {
	writtenData []byte
	writtenPath string
}

func (m *mockFileSystem) ReadFile(path string) ([]byte, error) { return nil, nil }
func (m *mockFileSystem) WriteFile(path string, data []byte) error {
	m.writtenPath = path
	m.writtenData = data
	return nil
}
func (m *mockFileSystem) Exists(path string) (bool, error) { return true, nil }
func (m *mockFileSystem) MkdirAll(path string) error       { return nil }
func (m *mockFileSystem) Remove(path string) error         { return nil }

// mockLogger for testing.
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, args ...interface{}) {}
func (m *mockLogger) Info(msg string, args ...interface{})  {}
func (m *mockLogger) Warn(msg string, args ...interface{})  {}
func (m *mockLogger) Error(msg string, args ...interface{}) {}
func (m *mockLogger) WithComponent(name string) ports.Logger { return m }

// createTestFrame creates a test image with specified dimensions.
func createTestFrame(width, height int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Gap != 1 {
		t.Errorf("DefaultOptions().Gap = %d, want 1", opts.Gap)
	}

	// Check that BorderColor matches DefaultBorderColor (#505050)
	r, g, b, a := opts.BorderColor.RGBA()
	dr, dg, db, da := DefaultBorderColor.RGBA()
	if r != dr || g != dg || b != db || a != da {
		t.Errorf("DefaultOptions().BorderColor = %v, want %v", opts.BorderColor, DefaultBorderColor)
	}
}

func TestOutputDimensions(t *testing.T) {
	tests := []struct {
		name        string
		leftWidth   int
		leftHeight  int
		rightWidth  int
		rightHeight int
		gap         int
		wantWidth   int
		wantHeight  int
	}{
		{
			name:        "same size videos with gap 1",
			leftWidth:   100,
			leftHeight:  200,
			rightWidth:  100,
			rightHeight: 200,
			gap:         1,
			wantWidth:   201, // 100 + 1 + 100
			wantHeight:  200,
		},
		{
			name:        "same size videos with gap 10",
			leftWidth:   100,
			leftHeight:  200,
			rightWidth:  100,
			rightHeight: 200,
			gap:         10,
			wantWidth:   210, // 100 + 10 + 100
			wantHeight:  200,
		},
		{
			name:        "different width videos",
			leftWidth:   100,
			leftHeight:  200,
			rightWidth:  150,
			rightHeight: 200,
			gap:         5,
			wantWidth:   255, // 100 + 5 + 150
			wantHeight:  200,
		},
		{
			name:        "different height videos (takes max)",
			leftWidth:   100,
			leftHeight:  200,
			rightWidth:  100,
			rightHeight: 300,
			gap:         1,
			wantWidth:   201,
			wantHeight:  300, // max(200, 300)
		},
		{
			name:        "zero gap",
			leftWidth:   100,
			leftHeight:  200,
			rightWidth:  100,
			rightHeight: 200,
			gap:         0,
			wantWidth:   200, // 100 + 0 + 100
			wantHeight:  200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock frames
			leftFrame := ports.VideoFrame{
				Image:       createTestFrame(tt.leftWidth, tt.leftHeight, color.White),
				TimestampMs: 0,
				Duration:    100,
			}
			rightFrame := ports.VideoFrame{
				Image:       createTestFrame(tt.rightWidth, tt.rightHeight, color.White),
				TimestampMs: 0,
				Duration:    100,
			}

			decoder := &mockDecoder{
				leftFrames:  []ports.VideoFrame{leftFrame},
				rightFrames: []ports.VideoFrame{rightFrame},
			}
			encoder := &mockEncoder{}
			fs := &mockFileSystem{}
			logger := &mockLogger{}

			opts := DefaultOptions()
			opts.Gap = tt.gap

			stage := New(decoder, encoder, fs, logger, opts)

			_, err := stage.Execute(context.Background(), Input{
				LeftPath:   "left.mp4",
				RightPath:  "right.mp4",
				OutputPath: "output.mp4",
			})
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if encoder.width != tt.wantWidth {
				t.Errorf("output width = %d, want %d", encoder.width, tt.wantWidth)
			}
			if encoder.height != tt.wantHeight {
				t.Errorf("output height = %d, want %d", encoder.height, tt.wantHeight)
			}
		})
	}
}

func TestBorderColor(t *testing.T) {
	tests := []struct {
		name        string
		borderColor color.Color
		gap         int
	}{
		{
			name:        "default border color",
			borderColor: DefaultBorderColor,
			gap:         5,
		},
		{
			name:        "white border",
			borderColor: color.White,
			gap:         5,
		},
		{
			name:        "red border",
			borderColor: color.RGBA{R: 255, G: 0, B: 0, A: 255},
			gap:         5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leftFrame := ports.VideoFrame{
				Image:       createTestFrame(100, 100, color.Black),
				TimestampMs: 0,
				Duration:    100,
			}
			rightFrame := ports.VideoFrame{
				Image:       createTestFrame(100, 100, color.Black),
				TimestampMs: 0,
				Duration:    100,
			}

			decoder := &mockDecoder{
				leftFrames:  []ports.VideoFrame{leftFrame},
				rightFrames: []ports.VideoFrame{rightFrame},
			}
			encoder := &mockEncoder{}
			fs := &mockFileSystem{}
			logger := &mockLogger{}

			opts := DefaultOptions()
			opts.Gap = tt.gap
			opts.BorderColor = tt.borderColor

			stage := New(decoder, encoder, fs, logger, opts)

			_, err := stage.Execute(context.Background(), Input{
				LeftPath:   "left.mp4",
				RightPath:  "right.mp4",
				OutputPath: "output.mp4",
			})
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			// Check that at least one frame was encoded
			if len(encoder.frames) == 0 {
				t.Fatal("no frames encoded")
			}

			// Check the border color in the gap area (between left and right video)
			frame := encoder.frames[0].(*image.RGBA)
			// The gap starts at x=100 (left video width) and ends at x=105 (100+gap)
			gapX := 100 + tt.gap/2 // middle of gap
			gapY := 50             // middle of height

			gotColor := frame.At(gapX, gapY)
			r1, g1, b1, a1 := gotColor.RGBA()
			r2, g2, b2, a2 := tt.borderColor.RGBA()

			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				t.Errorf("border color at (%d, %d) = %v, want %v", gapX, gapY, gotColor, tt.borderColor)
			}
		})
	}
}
