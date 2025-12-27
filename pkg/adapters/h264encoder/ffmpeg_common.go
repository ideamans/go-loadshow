// Package h264encoder provides H.264 video encoding using platform-native APIs.
// This file provides FFmpeg-based encoding as a fallback for all platforms.
package h264encoder

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/user/loadshow/pkg/ports"
)

// IsFFmpegAvailable checks if ffmpeg is available on the system.
func IsFFmpegAvailable() bool {
	_, err := FindFFmpeg()
	return err == nil
}

// FindFFmpeg searches for ffmpeg in PATH and common locations.
// Priority: 1) customFFmpegPath (set via SetFFmpegPath), 2) FFMPEG_PATH env, 3) PATH, 4) common locations
func FindFFmpeg() (string, error) {
	// Check custom path first (set via SetFFmpegPath)
	if customFFmpegPath != "" {
		if _, err := os.Stat(customFFmpegPath); err == nil {
			return customFFmpegPath, nil
		}
		return "", fmt.Errorf("%w: custom path %s not found", ErrFFmpegNotFound, customFFmpegPath)
	}

	// Check FFMPEG_PATH environment variable
	if envPath := os.Getenv("FFMPEG_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
		return "", fmt.Errorf("%w: FFMPEG_PATH %s not found", ErrFFmpegNotFound, envPath)
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
	} else if runtime.GOOS == "darwin" {
		commonPaths = []string{
			"/opt/homebrew/bin/ffmpeg",
			"/usr/local/bin/ffmpeg",
			"/usr/bin/ffmpeg",
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

// FFmpegEncoder implements H.264 encoding using ffmpeg external process.
// This can be used as a fallback on any platform.
type FFmpegEncoder struct {
	ffmpegPath string
	width      int
	height     int
	fps        float64
	opts       ports.EncoderOptions

	mu         sync.Mutex
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stderr     bytes.Buffer
	tempPath   string
	frameCount int
	closed     bool
}

// NewFFmpegEncoder creates a new FFmpeg-based H.264 encoder.
func NewFFmpegEncoder() *FFmpegEncoder {
	return &FFmpegEncoder{}
}

// Begin initializes the encoder.
func (e *FFmpegEncoder) Begin(width, height int, fps float64, opts ports.EncoderOptions) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Find ffmpeg
	ffmpegPath, err := FindFFmpeg()
	if err != nil {
		return err
	}
	e.ffmpegPath = ffmpegPath

	e.width = width
	e.height = height
	e.fps = fps
	e.opts = opts
	e.frameCount = 0
	e.closed = false

	// Create temporary output file
	tmpFile, err := os.CreateTemp("", "h264encode_*.mp4")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	e.tempPath = tmpFile.Name()
	tmpFile.Close()

	// Build ffmpeg arguments
	args := []string{
		"-y",             // Overwrite output
		"-f", "rawvideo", // Input format
		"-pix_fmt", "rgba", // Input pixel format
		"-s", fmt.Sprintf("%dx%d", width, height), // Input size
		"-r", fmt.Sprintf("%.2f", fps), // Input frame rate
		"-i", "pipe:0", // Read from stdin
		"-c:v", "libx264", // Use libx264
		"-preset", "fast", // Encoding preset
		"-pix_fmt", "yuv420p", // Output pixel format
	}

	// Add quality settings
	if opts.Quality > 0 && opts.Quality <= 63 {
		// Convert our 0-63 scale to x264's CRF (0-51)
		crf := opts.Quality * 51 / 63
		if crf > 51 {
			crf = 51
		}
		args = append(args, "-crf", fmt.Sprintf("%d", crf))
	} else {
		args = append(args, "-crf", "23") // Default quality
	}

	// Add bitrate if specified
	if opts.Bitrate > 0 {
		args = append(args, "-b:v", fmt.Sprintf("%dk", opts.Bitrate))
	}

	// Profile for compatibility
	args = append(args,
		"-profile:v", "baseline",
		"-level", "3.1",
		"-movflags", "+faststart",
		e.tempPath,
	)

	// Start ffmpeg
	e.cmd = exec.Command(e.ffmpegPath, args...)
	e.cmd.Stderr = &e.stderr

	stdin, err := e.cmd.StdinPipe()
	if err != nil {
		os.Remove(e.tempPath)
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	e.stdin = stdin

	if err := e.cmd.Start(); err != nil {
		os.Remove(e.tempPath)
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	return nil
}

// EncodeFrame encodes a single frame.
func (e *FFmpegEncoder) EncodeFrame(img image.Image, timestampMs int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stdin == nil || e.closed {
		return ErrNotInitialized
	}

	// Convert image to RGBA
	bounds := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, e.width, e.height))
	draw.Draw(rgba, rgba.Bounds(), img, bounds.Min, draw.Src)

	// Write raw RGBA data to ffmpeg stdin
	_, err := e.stdin.Write(rgba.Pix)
	if err != nil {
		return fmt.Errorf("failed to write frame: %w", err)
	}

	e.frameCount++
	return nil
}

// End finalizes encoding and returns the MP4 data.
func (e *FFmpegEncoder) End() ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stdin == nil || e.closed {
		return nil, ErrNotInitialized
	}

	// Close stdin to signal end of input
	e.stdin.Close()
	e.stdin = nil
	e.closed = true

	// Wait for ffmpeg to finish
	if err := e.cmd.Wait(); err != nil {
		stderrOutput := e.stderr.String()
		return nil, fmt.Errorf("ffmpeg encoding failed: %w\nstderr: %s", err, stderrOutput)
	}

	// Read the output file
	data, err := os.ReadFile(e.tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output: %w", err)
	}

	// Clean up temp file
	os.Remove(e.tempPath)
	e.tempPath = ""

	return data, nil
}

// Ensure FFmpegEncoder implements ports.VideoEncoder
var _ ports.VideoEncoder = (*FFmpegEncoder)(nil)
