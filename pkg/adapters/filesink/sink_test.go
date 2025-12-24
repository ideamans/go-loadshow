package filesink

import (
	"image"
	"testing"

	"github.com/user/loadshow/pkg/mocks"
	"github.com/user/loadshow/pkg/ports"
)

func TestSink_Enabled(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{}
	sink := New("/debug", fs, renderer)

	if !sink.Enabled() {
		t.Error("expected Enabled to return true")
	}
}

func TestSink_SaveLayoutJSON(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{}
	sink := New("/debug", fs, renderer)

	data := []byte(`{"test": true}`)
	err := sink.SaveLayoutJSON(data)
	if err != nil {
		t.Fatalf("SaveLayoutJSON failed: %v", err)
	}

	saved, ok := fs.GetFile("/debug/layout.json")
	if !ok {
		t.Error("expected file to be saved")
	}
	if string(saved) != string(data) {
		t.Errorf("expected %q, got %q", data, saved)
	}
}

func TestSink_SaveLayoutSVG(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{}
	sink := New("/debug", fs, renderer)

	data := []byte(`<svg></svg>`)
	err := sink.SaveLayoutSVG(data)
	if err != nil {
		t.Fatalf("SaveLayoutSVG failed: %v", err)
	}

	saved, ok := fs.GetFile("/debug/layout.svg")
	if !ok {
		t.Error("expected file to be saved")
	}
	if string(saved) != string(data) {
		t.Errorf("expected %q, got %q", data, saved)
	}
}

func TestSink_SaveRecordingJSON(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{}
	sink := New("/debug", fs, renderer)

	data := []byte(`{"duration": 1000}`)
	err := sink.SaveRecordingJSON(data)
	if err != nil {
		t.Fatalf("SaveRecordingJSON failed: %v", err)
	}

	_, ok := fs.GetFile("/debug/recording.json")
	if !ok {
		t.Error("expected file to be saved")
	}
}

func TestSink_SaveRawFrame(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{}
	sink := New("/debug", fs, renderer)

	data := []byte{0xFF, 0xD8, 0xFF} // JPEG header
	err := sink.SaveRawFrame(0, data)
	if err != nil {
		t.Fatalf("SaveRawFrame failed: %v", err)
	}

	_, ok := fs.GetFile("/debug/frames/raw/frame-0000.jpg")
	if !ok {
		t.Error("expected file to be saved")
	}
}

func TestSink_SaveBanner(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{
		EncodeImageFunc: func(img image.Image, format ports.ImageFormat, quality int) ([]byte, error) {
			return []byte{0x89, 0x50, 0x4E, 0x47}, nil // PNG header
		},
	}
	sink := New("/debug", fs, renderer)

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	err := sink.SaveBanner(img)
	if err != nil {
		t.Fatalf("SaveBanner failed: %v", err)
	}

	_, ok := fs.GetFile("/debug/banner.png")
	if !ok {
		t.Error("expected file to be saved")
	}
}

func TestSink_SaveComposedFrame(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{
		EncodeImageFunc: func(img image.Image, format ports.ImageFormat, quality int) ([]byte, error) {
			return []byte{0x89, 0x50, 0x4E, 0x47}, nil
		},
	}
	sink := New("/debug", fs, renderer)

	img := image.NewRGBA(image.Rect(0, 0, 512, 640))
	err := sink.SaveComposedFrame(5, img)
	if err != nil {
		t.Fatalf("SaveComposedFrame failed: %v", err)
	}

	_, ok := fs.GetFile("/debug/frames/composed/frame-0005.png")
	if !ok {
		t.Error("expected file to be saved")
	}
}

func TestSink_MultipleRawFrames(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{}
	sink := New("/debug", fs, renderer)

	for i := 0; i < 10; i++ {
		err := sink.SaveRawFrame(i, []byte{0xFF})
		if err != nil {
			t.Fatalf("SaveRawFrame %d failed: %v", i, err)
		}
	}

	// Check all files exist
	files := fs.GetAllFiles()
	expectedCount := 10
	count := 0
	for path := range files {
		if len(path) > 0 {
			count++
		}
	}

	if count != expectedCount {
		t.Errorf("expected %d files, got %d", expectedCount, count)
	}
}
