// Package e2e contains end-to-end tests for the loadshow CLI.
// This package has no CGO dependencies so it can run with pre-built binaries.
package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const testURL = "https://dummy-ec-site.ideamans.com/"

// getBinaryName returns the test binary name with platform-specific extension
func getBinaryName() string {
	if runtime.GOOS == "windows" {
		return "loadshow-test.exe"
	}
	return "loadshow-test"
}

// getBinaryPath returns the path to execute the test binary
// If LOADSHOW_BINARY env var is set, use that instead (for CI with pre-built binaries)
func getBinaryPath() string {
	if path := os.Getenv("LOADSHOW_BINARY"); path != "" {
		return path
	}
	if runtime.GOOS == "windows" {
		return ".\\loadshow-test.exe"
	}
	return "./loadshow-test"
}

// shouldBuildBinary returns true if we need to build the binary (no pre-built binary provided)
func shouldBuildBinary() bool {
	return os.Getenv("LOADSHOW_BINARY") == ""
}

// TestRecordCommand tests the record subcommand with a real website
func TestRecordCommand(t *testing.T) {
	if os.Getenv("LOADSHOW_E2E") != "1" {
		t.Skip("Skipping E2E test (set LOADSHOW_E2E=1 to run)")
	}

	// Build the CLI if no pre-built binary is provided
	if shouldBuildBinary() {
		buildCmd := exec.Command("go", "build", "-o", getBinaryName(), "./cmd/loadshow")
		buildCmd.Dir = getProjectRoot(t)
		if out, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build CLI: %v\n%s", err, out)
		}
		defer os.Remove(filepath.Join(getProjectRoot(t), getBinaryName()))
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "loadshow-e2e-*.mp4")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Run the record command (flags must come before URL argument in urfave/cli)
	cmd := exec.Command(
		getBinaryPath(),
		"record",
		"-o", tmpFile.Name(),
		"-p", "mobile",
		testURL,
	)
	cmd.Dir = getProjectRoot(t)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Record command failed: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// Verify output file
	info, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatalf("Output file not found: %v", err)
	}

	// Check file size is reasonable (at least 10KB for a real video)
	if info.Size() < 10*1024 {
		t.Errorf("Output file too small: %d bytes", info.Size())
	}

	// Read and verify video content
	videoData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	// Verify MP4 signature
	if len(videoData) < 8 || string(videoData[4:8]) != "ftyp" {
		t.Error("Invalid MP4 file")
	}

	t.Logf("Video created: %d bytes", info.Size())
}

// TestRecordDesktopPreset tests the desktop preset
func TestRecordDesktopPreset(t *testing.T) {
	if os.Getenv("LOADSHOW_E2E") != "1" {
		t.Skip("Skipping E2E test (set LOADSHOW_E2E=1 to run)")
	}

	// Build the CLI if no pre-built binary is provided
	if shouldBuildBinary() {
		buildCmd := exec.Command("go", "build", "-o", getBinaryName(), "./cmd/loadshow")
		buildCmd.Dir = getProjectRoot(t)
		if out, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build CLI: %v\n%s", err, out)
		}
		defer os.Remove(filepath.Join(getProjectRoot(t), getBinaryName()))
	}

	tmpFile, err := os.CreateTemp("", "loadshow-e2e-desktop-*.mp4")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	cmd := exec.Command(
		getBinaryPath(),
		"record",
		"-o", tmpFile.Name(),
		"-p", "desktop",
		testURL,
	)
	cmd.Dir = getProjectRoot(t)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Record command failed: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// Verify output
	info, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatalf("Output file not found: %v", err)
	}

	if info.Size() < 10*1024 {
		t.Errorf("Output file too small: %d bytes", info.Size())
	}

	t.Logf("Desktop preset video: %d bytes", info.Size())
}

// TestRecordWithCustomDimensions tests custom width/height
func TestRecordWithCustomDimensions(t *testing.T) {
	if os.Getenv("LOADSHOW_E2E") != "1" {
		t.Skip("Skipping E2E test (set LOADSHOW_E2E=1 to run)")
	}

	if shouldBuildBinary() {
		buildCmd := exec.Command("go", "build", "-o", getBinaryName(), "./cmd/loadshow")
		buildCmd.Dir = getProjectRoot(t)
		if out, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build CLI: %v\n%s", err, out)
		}
		defer os.Remove(filepath.Join(getProjectRoot(t), getBinaryName()))
	}

	tmpFile, err := os.CreateTemp("", "loadshow-e2e-custom-*.mp4")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	cmd := exec.Command(
		getBinaryPath(),
		"record",
		"-o", tmpFile.Name(),
		"-W", "320",
		"-H", "400",
		"-c", "2",
		testURL,
	)
	cmd.Dir = getProjectRoot(t)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Record command failed: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	info, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatalf("Output file not found: %v", err)
	}

	t.Logf("Custom dimensions video: %d bytes", info.Size())

	// Verify MP4 file is valid
	videoData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	if len(videoData) < 8 || string(videoData[4:8]) != "ftyp" {
		t.Error("Invalid MP4 file")
	}
}

// TestRecordWithDebugOutput tests debug output
func TestRecordWithDebugOutput(t *testing.T) {
	if os.Getenv("LOADSHOW_E2E") != "1" {
		t.Skip("Skipping E2E test (set LOADSHOW_E2E=1 to run)")
	}

	if shouldBuildBinary() {
		buildCmd := exec.Command("go", "build", "-o", getBinaryName(), "./cmd/loadshow")
		buildCmd.Dir = getProjectRoot(t)
		if out, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build CLI: %v\n%s", err, out)
		}
		defer os.Remove(filepath.Join(getProjectRoot(t), getBinaryName()))
	}

	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "loadshow-e2e-debug-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "output.mp4")
	debugDir := filepath.Join(tmpDir, "debug")

	cmd := exec.Command(
		getBinaryPath(),
		"record",
		"-o", outputPath,
		"-p", "mobile",
		"-d",
		"--debug-dir", debugDir,
		testURL,
	)
	cmd.Dir = getProjectRoot(t)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Record command failed: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// Verify debug output
	if _, err := os.Stat(filepath.Join(debugDir, "layout.json")); os.IsNotExist(err) {
		t.Error("Expected layout.json in debug output")
	}

	// Check for raw frames
	entries, err := os.ReadDir(debugDir)
	if err != nil {
		t.Fatalf("Failed to read debug dir: %v", err)
	}

	hasRawFrames := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "raw_") {
			hasRawFrames = true
			break
		}
	}

	if !hasRawFrames {
		t.Log("Warning: no raw frames in debug output")
	}

	t.Logf("Debug output created with %d files", len(entries))
}

// TestVersionCommand tests the version flag
func TestVersionCommand(t *testing.T) {
	if os.Getenv("LOADSHOW_E2E") != "1" {
		t.Skip("Skipping E2E test (set LOADSHOW_E2E=1 to run)")
	}

	if shouldBuildBinary() {
		buildCmd := exec.Command("go", "build", "-o", getBinaryName(), "./cmd/loadshow")
		buildCmd.Dir = getProjectRoot(t)
		if out, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build CLI: %v\n%s", err, out)
		}
		defer os.Remove(filepath.Join(getProjectRoot(t), getBinaryName()))
	}

	// urfave/cli uses --version flag instead of version subcommand
	cmd := exec.Command(getBinaryPath(), "--version")
	cmd.Dir = getProjectRoot(t)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Version command failed: %v", err)
	}

	if !strings.Contains(string(out), "loadshow version") {
		t.Errorf("Unexpected version output: %s", out)
	}
}

// TestJuxtaposeCommand tests the juxtapose subcommand
func TestJuxtaposeCommand(t *testing.T) {
	if os.Getenv("LOADSHOW_E2E") != "1" {
		t.Skip("Skipping E2E test (set LOADSHOW_E2E=1 to run)")
	}

	if shouldBuildBinary() {
		buildCmd := exec.Command("go", "build", "-o", getBinaryName(), "./cmd/loadshow")
		buildCmd.Dir = getProjectRoot(t)
		if out, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build CLI: %v\n%s", err, out)
		}
		defer os.Remove(filepath.Join(getProjectRoot(t), getBinaryName()))
	}

	// First, create two videos to juxtapose
	tmpDir, err := os.MkdirTemp("", "loadshow-e2e-juxta-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	leftPath := filepath.Join(tmpDir, "left.mp4")
	rightPath := filepath.Join(tmpDir, "right.mp4")
	outputPath := filepath.Join(tmpDir, "output.mp4")

	// Create left video
	cmd := exec.Command(
		getBinaryPath(),
		"record",
		"-o", leftPath,
		"-p", "mobile",
		testURL,
	)
	cmd.Dir = getProjectRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create left video: %v\n%s", err, out)
	}

	// Create right video
	cmd = exec.Command(
		getBinaryPath(),
		"record",
		"-o", rightPath,
		"-p", "desktop",
		testURL,
	)
	cmd.Dir = getProjectRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create right video: %v\n%s", err, out)
	}

	// Run juxtapose
	cmd = exec.Command(
		getBinaryPath(),
		"juxtapose",
		"-o", outputPath,
		leftPath,
		rightPath,
	)
	cmd.Dir = getProjectRoot(t)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Juxtapose command failed: %v\n%s", err, out)
	}

	// Verify output file was created
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Output file not found: %v", err)
	}

	// Check file size is reasonable (at least 10KB for a real video)
	if info.Size() < 10*1024 {
		t.Errorf("Output file too small: %d bytes", info.Size())
	}

	t.Logf("Juxtapose video created: %d bytes", info.Size())
}

// TestRecordWithCodecH264 tests recording with H.264 codec (default)
func TestRecordWithCodecH264(t *testing.T) {
	if os.Getenv("LOADSHOW_E2E") != "1" {
		t.Skip("Skipping E2E test (set LOADSHOW_E2E=1 to run)")
	}

	if shouldBuildBinary() {
		buildCmd := exec.Command("go", "build", "-o", getBinaryName(), "./cmd/loadshow")
		buildCmd.Dir = getProjectRoot(t)
		if out, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build CLI: %v\n%s", err, out)
		}
		defer os.Remove(filepath.Join(getProjectRoot(t), getBinaryName()))
	}

	tmpFile, err := os.CreateTemp("", "loadshow-e2e-h264-*.mp4")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Use explicit --codec h264 flag
	cmd := exec.Command(
		getBinaryPath(),
		"record",
		"-o", tmpFile.Name(),
		"-p", "mobile",
		"--codec", "h264",
		testURL,
	)
	cmd.Dir = getProjectRoot(t)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// If H.264 is not available, it should fall back to AV1
		if strings.Contains(stderr.String(), "falling back to AV1") {
			t.Log("H.264 not available, fell back to AV1")
		} else {
			t.Fatalf("Record command failed: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
		}
	}

	// Verify output file
	info, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatalf("Output file not found: %v", err)
	}

	if info.Size() < 10*1024 {
		t.Errorf("Output file too small: %d bytes", info.Size())
	}

	// Check if H.264 codec was used (look for log output)
	output := stdout.String() + stderr.String()
	if strings.Contains(output, "H.264 codec") {
		t.Log("Successfully recorded with H.264 codec")
	} else if strings.Contains(output, "AV1") {
		t.Log("Recorded with AV1 codec (fallback)")
	}

	t.Logf("H.264 video created: %d bytes", info.Size())
}

// TestRecordWithCodecAV1 tests recording with AV1 codec (explicit)
func TestRecordWithCodecAV1(t *testing.T) {
	if os.Getenv("LOADSHOW_E2E") != "1" {
		t.Skip("Skipping E2E test (set LOADSHOW_E2E=1 to run)")
	}

	if shouldBuildBinary() {
		buildCmd := exec.Command("go", "build", "-o", getBinaryName(), "./cmd/loadshow")
		buildCmd.Dir = getProjectRoot(t)
		if out, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build CLI: %v\n%s", err, out)
		}
		defer os.Remove(filepath.Join(getProjectRoot(t), getBinaryName()))
	}

	tmpFile, err := os.CreateTemp("", "loadshow-e2e-av1-*.mp4")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Use explicit --codec av1 flag
	cmd := exec.Command(
		getBinaryPath(),
		"record",
		"-o", tmpFile.Name(),
		"-p", "mobile",
		"--codec", "av1",
		testURL,
	)
	cmd.Dir = getProjectRoot(t)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Record command failed: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// Verify output file
	info, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatalf("Output file not found: %v", err)
	}

	if info.Size() < 10*1024 {
		t.Errorf("Output file too small: %d bytes", info.Size())
	}

	// Check if AV1 codec was used
	output := stdout.String() + stderr.String()
	if strings.Contains(output, "AV1 codec") {
		t.Log("Successfully recorded with AV1 codec")
	}

	t.Logf("AV1 video created: %d bytes", info.Size())
}

// TestJuxtaposeWithCodec tests juxtapose with codec options
func TestJuxtaposeWithCodec(t *testing.T) {
	if os.Getenv("LOADSHOW_E2E") != "1" {
		t.Skip("Skipping E2E test (set LOADSHOW_E2E=1 to run)")
	}

	if shouldBuildBinary() {
		buildCmd := exec.Command("go", "build", "-o", getBinaryName(), "./cmd/loadshow")
		buildCmd.Dir = getProjectRoot(t)
		if out, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build CLI: %v\n%s", err, out)
		}
		defer os.Remove(filepath.Join(getProjectRoot(t), getBinaryName()))
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "loadshow-e2e-juxta-codec-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	leftPath := filepath.Join(tmpDir, "left.mp4")
	rightPath := filepath.Join(tmpDir, "right.mp4")
	outputPath := filepath.Join(tmpDir, "output.mp4")

	// Create left video (using default codec - H.264 if available)
	cmd := exec.Command(
		getBinaryPath(),
		"record",
		"-o", leftPath,
		"-p", "mobile",
		testURL,
	)
	cmd.Dir = getProjectRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create left video: %v\n%s", err, out)
	}

	// Create right video
	cmd = exec.Command(
		getBinaryPath(),
		"record",
		"-o", rightPath,
		"-p", "desktop",
		testURL,
	)
	cmd.Dir = getProjectRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create right video: %v\n%s", err, out)
	}

	// Run juxtapose with explicit codec
	cmd = exec.Command(
		getBinaryPath(),
		"juxtapose",
		"-o", outputPath,
		"--codec", "h264",
		leftPath,
		rightPath,
	)
	cmd.Dir = getProjectRoot(t)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// If H.264 not available, should fall back
		if !strings.Contains(stderr.String(), "falling back") {
			t.Fatalf("Juxtapose command failed: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
		}
	}

	// Verify output file
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Output file not found: %v", err)
	}

	if info.Size() < 10*1024 {
		t.Errorf("Output file too small: %d bytes", info.Size())
	}

	t.Logf("Juxtapose with codec option: %d bytes", info.Size())
}

// TestRecordWithProxy tests recording with proxy option
func TestRecordWithProxy(t *testing.T) {
	if os.Getenv("LOADSHOW_E2E") != "1" {
		t.Skip("Skipping E2E test (set LOADSHOW_E2E=1 to run)")
	}

	// This test just verifies the proxy option is accepted
	// Actual proxy testing would require a proxy server

	if shouldBuildBinary() {
		buildCmd := exec.Command("go", "build", "-o", getBinaryName(), "./cmd/loadshow")
		buildCmd.Dir = getProjectRoot(t)
		if out, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build CLI: %v\n%s", err, out)
		}
		defer os.Remove(filepath.Join(getProjectRoot(t), getBinaryName()))
	}

	// Just verify the help shows the proxy option
	cmd := exec.Command(getBinaryPath(), "record", "--help")
	cmd.Dir = getProjectRoot(t)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	if !strings.Contains(string(out), "--proxy-server") {
		t.Error("Expected --proxy-server option in help")
	}

	if !strings.Contains(string(out), "--ignore-https-errors") {
		t.Error("Expected --ignore-https-errors option in help")
	}

	if !strings.Contains(string(out), "--codec") {
		t.Error("Expected --codec option in help")
	}
}

// getProjectRoot returns the project root directory
func getProjectRoot(t *testing.T) string {
	// Start from current working directory and find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root (go.mod)")
		}
		dir = parent
	}
}
