//go:build linux

package h264decoder

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

// ffmpegDecoder implements H.264 decoding using ffmpeg external process.
type ffmpegDecoder struct {
	ffmpegPath string
	mu         sync.Mutex
	width      int
	height     int
	initialized bool
}

func newPlatformDecoder() platformDecoder {
	return &ffmpegDecoder{}
}

// findFFmpeg searches for ffmpeg in PATH and common locations.
// If customFFmpegPath is set, it uses that path instead.
func findFFmpeg() (string, error) {
	// Check custom path first (set via SetFFmpegPath)
	if customFFmpegPath != "" {
		if _, err := os.Stat(customFFmpegPath); err == nil {
			return customFFmpegPath, nil
		}
		return "", fmt.Errorf("%w: custom path %s not found", ErrFFmpegNotFound, customFFmpegPath)
	}

	// Check PATH
	execName := "ffmpeg"
	if runtime.GOOS == "windows" {
		execName = "ffmpeg.exe"
	}

	path, err := exec.LookPath(execName)
	if err == nil {
		return path, nil
	}

	// Check common locations
	var commonPaths []string
	if runtime.GOOS == "windows" {
		commonPaths = []string{
			`C:\ffmpeg\bin\ffmpeg.exe`,
			`C:\Program Files\ffmpeg\bin\ffmpeg.exe`,
			`C:\Program Files (x86)\ffmpeg\bin\ffmpeg.exe`,
		}
	} else {
		commonPaths = []string{
			"/usr/bin/ffmpeg",
			"/usr/local/bin/ffmpeg",
			"/opt/homebrew/bin/ffmpeg",
			"/snap/bin/ffmpeg",
		}
	}

	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", ErrFFmpegNotFound
}

// checkPlatformAvailability checks if ffmpeg is available on the system.
func checkPlatformAvailability() bool {
	_, err := findFFmpeg()
	return err == nil
}

func (d *ffmpegDecoder) init() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	ffmpegPath, err := findFFmpeg()
	if err != nil {
		return err
	}
	d.ffmpegPath = ffmpegPath
	d.initialized = true
	return nil
}

// decodeFrame decodes a single H.264 frame from Annex B format using ffmpeg.
// This creates a temporary file, writes the frame data, and uses ffmpeg to decode.
func (d *ffmpegDecoder) decodeFrame(data []byte) (image.Image, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.initialized {
		return nil, ErrNotInitialized
	}

	if len(data) == 0 {
		return nil, ErrDecodeFailed
	}

	// Create temp file for input
	inputFile, err := os.CreateTemp("", "h264frame_*.h264")
	if err != nil {
		return nil, fmt.Errorf("create input temp file: %w", err)
	}
	inputPath := inputFile.Name()
	defer os.Remove(inputPath)

	// Write frame data
	if _, err := inputFile.Write(data); err != nil {
		inputFile.Close()
		return nil, fmt.Errorf("write frame data: %w", err)
	}
	inputFile.Close()

	// Create temp file for output
	outputFile, err := os.CreateTemp("", "h264frame_*.png")
	if err != nil {
		return nil, fmt.Errorf("create output temp file: %w", err)
	}
	outputPath := outputFile.Name()
	outputFile.Close()
	defer os.Remove(outputPath)

	// Run ffmpeg to decode
	var stderr bytes.Buffer
	cmd := exec.Command(d.ffmpegPath,
		"-y",
		"-f", "h264",
		"-i", inputPath,
		"-frames:v", "1",
		"-f", "image2",
		outputPath,
	)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg decode failed: %w\nstderr: %s", err, stderr.String())
	}

	// Read decoded image
	imgFile, err := os.Open(outputPath)
	if err != nil {
		return nil, fmt.Errorf("open decoded image: %w", err)
	}
	defer imgFile.Close()

	img, err := png.Decode(imgFile)
	if err != nil {
		return nil, fmt.Errorf("decode png: %w", err)
	}

	return img, nil
}

func (d *ffmpegDecoder) close() {
	d.initialized = false
}
