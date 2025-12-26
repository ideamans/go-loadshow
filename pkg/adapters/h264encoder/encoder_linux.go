//go:build linux

package h264encoder

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/user/loadshow/pkg/ports"
)

// ffmpegEncoder implements H.264 encoding using ffmpeg external process on Linux.
type ffmpegEncoder struct {
	ffmpegPath string
	width      int
	height     int
	fps        float64
	opts       ports.EncoderOptions

	mu         sync.Mutex
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	outputFile *os.File
	tempPath   string
	frameCount int
	firstFrame bool
}

func newPlatformEncoder() platformEncoder {
	return &ffmpegEncoder{}
}

// findFFmpeg searches for ffmpeg in PATH
func findFFmpeg() (string, error) {
	// Check PATH
	path, err := exec.LookPath("ffmpeg")
	if err == nil {
		return path, nil
	}

	// Check common locations
	commonPaths := []string{
		"/usr/bin/ffmpeg",
		"/usr/local/bin/ffmpeg",
		"/opt/homebrew/bin/ffmpeg",
	}

	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", ErrFFmpegNotFound
}

func (e *ffmpegEncoder) init(width, height int, fps float64, opts ports.EncoderOptions) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Find ffmpeg
	ffmpegPath, err := findFFmpeg()
	if err != nil {
		return err
	}
	e.ffmpegPath = ffmpegPath

	e.width = width
	e.height = height
	e.fps = fps
	e.opts = opts
	e.frameCount = 0
	e.firstFrame = true

	// Create temporary output file
	tmpFile, err := os.CreateTemp("", "h264encode_*.mp4")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	e.tempPath = tmpFile.Name()
	tmpFile.Close()

	// Build ffmpeg arguments
	args := []string{
		"-y",                    // Overwrite output
		"-f", "rawvideo",        // Input format
		"-pix_fmt", "rgba",      // Input pixel format
		"-s", fmt.Sprintf("%dx%d", width, height), // Input size
		"-r", fmt.Sprintf("%.2f", fps),            // Input frame rate
		"-i", "pipe:0",          // Read from stdin
		"-c:v", "libx264",       // Use libx264
		"-preset", "fast",       // Encoding preset
		"-pix_fmt", "yuv420p",   // Output pixel format
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
	e.cmd.Stderr = io.Discard // Suppress ffmpeg output

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

func (e *ffmpegEncoder) encodeFrame(img image.Image, timestampMs int) ([]encodedFrame, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stdin == nil {
		return nil, ErrNotInitialized
	}

	// Convert image to RGBA
	bounds := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, e.width, e.height))
	draw.Draw(rgba, rgba.Bounds(), img, bounds.Min, draw.Src)

	// Write raw RGBA data to ffmpeg stdin
	_, err := e.stdin.Write(rgba.Pix)
	if err != nil {
		return nil, fmt.Errorf("failed to write frame: %w", err)
	}

	e.frameCount++
	return nil, nil // ffmpeg handles everything internally
}

func (e *ffmpegEncoder) flush() ([]encodedFrame, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stdin == nil {
		return nil, nil
	}

	// Close stdin to signal end of input
	e.stdin.Close()
	e.stdin = nil

	// Wait for ffmpeg to finish
	if err := e.cmd.Wait(); err != nil {
		return nil, fmt.Errorf("ffmpeg encoding failed: %w", err)
	}

	// Read the output file
	data, err := os.ReadFile(e.tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output: %w", err)
	}

	// Parse MP4 to extract H.264 frames (we need raw NAL units for our MP4 builder)
	frames, err := e.extractFramesFromMP4(data)
	if err != nil {
		// Fallback: return data as-is wrapped in a single "frame"
		// The caller will need to handle this differently
		return []encodedFrame{{
			data:        data,
			timestampUs: 0,
			isKeyframe:  true,
		}}, nil
	}

	return frames, nil
}

// extractFramesFromMP4 extracts H.264 NAL units from MP4 container
func (e *ffmpegEncoder) extractFramesFromMP4(data []byte) ([]encodedFrame, error) {
	// For simplicity, we'll return the raw MP4 data
	// The MP4 building in the main encoder will detect this and skip re-muxing
	// This is a simplification - in production we'd properly parse the MP4

	// For now, mark this as a complete MP4 by returning a special marker
	return nil, fmt.Errorf("MP4 passthrough")
}

func (e *ffmpegEncoder) close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stdin != nil {
		e.stdin.Close()
		e.stdin = nil
	}

	if e.cmd != nil && e.cmd.Process != nil {
		e.cmd.Process.Kill()
		e.cmd = nil
	}

	if e.tempPath != "" {
		os.Remove(e.tempPath)
		e.tempPath = ""
	}
}

// getOutputMP4 returns the complete MP4 file produced by ffmpeg.
// This is used when ffmpeg produces the final output directly.
func (e *ffmpegEncoder) getOutputMP4() ([]byte, error) {
	if e.tempPath == "" {
		return nil, ErrNotInitialized
	}

	return os.ReadFile(e.tempPath)
}

// Encoder extension for Linux - override buildMP4 to use ffmpeg's output directly
type linuxEncoderExt interface {
	getOutputMP4() ([]byte, error)
}

// IsFFmpegEncoder returns true if this is an ffmpeg-based encoder
func IsFFmpegEncoder(enc platformEncoder) bool {
	_, ok := enc.(*ffmpegEncoder)
	return ok
}

// GetFFmpegOutput returns the complete MP4 from ffmpeg encoder
func GetFFmpegOutput(enc platformEncoder) ([]byte, error) {
	if fe, ok := enc.(*ffmpegEncoder); ok {
		return fe.getOutputMP4()
	}
	return nil, fmt.Errorf("not an ffmpeg encoder")
}

// init registers a hook to override MP4 building on Linux
func init() {
	// This is handled in the main encoder's End() method
}

// Helper to check if we should use direct ffmpeg output
func useDirectFFmpegOutput() bool {
	return true // Always use ffmpeg's output on Linux
}

// readMP4Frames is a placeholder - in production this would parse MP4
func readMP4Frames(r io.Reader) ([]encodedFrame, error) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, err
	}

	// Return as single frame (MP4 passthrough)
	return []encodedFrame{{
		data:        buf.Bytes(),
		timestampUs: 0,
		isKeyframe:  true,
	}}, nil
}
