package filesink

import (
	"image"
	"path/filepath"
	"testing"

	"github.com/user/loadshow/pkg/mocks"
	"github.com/user/loadshow/pkg/ports"
)

// testBaseDir is a platform-independent base directory for tests
var testBaseDir = filepath.Join("debug")

func TestSink_Enabled(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{}
	sink := New(testBaseDir, fs, renderer)

	if !sink.Enabled() {
		t.Error("expected Enabled to return true")
	}
}

func TestSink_SaveLayoutJSON(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{}
	sink := New(testBaseDir, fs, renderer)

	data := []byte(`{"test": true}`)
	err := sink.SaveLayoutJSON(data)
	if err != nil {
		t.Fatalf("SaveLayoutJSON failed: %v", err)
	}

	expectedPath := filepath.Join(testBaseDir, "layout.json")
	saved, ok := fs.GetFile(expectedPath)
	if !ok {
		t.Errorf("expected file to be saved at %s", expectedPath)
	}
	if string(saved) != string(data) {
		t.Errorf("expected %q, got %q", data, saved)
	}
}

func TestSink_SaveLayoutSVG(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{}
	sink := New(testBaseDir, fs, renderer)

	data := []byte(`<svg></svg>`)
	err := sink.SaveLayoutSVG(data)
	if err != nil {
		t.Fatalf("SaveLayoutSVG failed: %v", err)
	}

	expectedPath := filepath.Join(testBaseDir, "layout.svg")
	saved, ok := fs.GetFile(expectedPath)
	if !ok {
		t.Errorf("expected file to be saved at %s", expectedPath)
	}
	if string(saved) != string(data) {
		t.Errorf("expected %q, got %q", data, saved)
	}
}

func TestSink_SaveRecordingJSON(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{}
	sink := New(testBaseDir, fs, renderer)

	data := []byte(`{"duration": 1000}`)
	err := sink.SaveRecordingJSON(data)
	if err != nil {
		t.Fatalf("SaveRecordingJSON failed: %v", err)
	}

	expectedPath := filepath.Join(testBaseDir, "recording.json")
	_, ok := fs.GetFile(expectedPath)
	if !ok {
		t.Errorf("expected file to be saved at %s", expectedPath)
	}
}

func TestSink_SaveRawFrame(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{}
	sink := New(testBaseDir, fs, renderer)

	data := []byte{0xFF, 0xD8, 0xFF} // JPEG header
	err := sink.SaveRawFrame(0, data)
	if err != nil {
		t.Fatalf("SaveRawFrame failed: %v", err)
	}

	expectedPath := filepath.Join(testBaseDir, "frames", "raw", "frame-0000.jpg")
	_, ok := fs.GetFile(expectedPath)
	if !ok {
		t.Errorf("expected file to be saved at %s", expectedPath)
	}
}

func TestSink_SaveBanner(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{
		EncodeImageFunc: func(img image.Image, format ports.ImageFormat, quality int) ([]byte, error) {
			return []byte{0x89, 0x50, 0x4E, 0x47}, nil // PNG header
		},
	}
	sink := New(testBaseDir, fs, renderer)

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	err := sink.SaveBanner(img)
	if err != nil {
		t.Fatalf("SaveBanner failed: %v", err)
	}

	expectedPath := filepath.Join(testBaseDir, "banner.png")
	_, ok := fs.GetFile(expectedPath)
	if !ok {
		t.Errorf("expected file to be saved at %s", expectedPath)
	}
}

func TestSink_SaveComposedFrame(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{
		EncodeImageFunc: func(img image.Image, format ports.ImageFormat, quality int) ([]byte, error) {
			return []byte{0x89, 0x50, 0x4E, 0x47}, nil
		},
	}
	sink := New(testBaseDir, fs, renderer)

	img := image.NewRGBA(image.Rect(0, 0, 512, 640))
	err := sink.SaveComposedFrame(5, img)
	if err != nil {
		t.Fatalf("SaveComposedFrame failed: %v", err)
	}

	expectedPath := filepath.Join(testBaseDir, "frames", "composed", "frame-0005.png")
	_, ok := fs.GetFile(expectedPath)
	if !ok {
		t.Errorf("expected file to be saved at %s", expectedPath)
	}
}

func TestSink_MultipleRawFrames(t *testing.T) {
	fs := mocks.NewFileSystem()
	renderer := &mocks.Renderer{}
	sink := New(testBaseDir, fs, renderer)

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
