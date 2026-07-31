package chromebrowser

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveChromePath_ExplicitPath(t *testing.T) {
	// Explicit path should always be returned
	result := ResolveChromePath("/custom/path/to/chrome")
	if result != "/custom/path/to/chrome" {
		t.Errorf("expected explicit path to be returned, got %s", result)
	}
}

func TestResolveChromePath_EnvVar(t *testing.T) {
	// Set CHROME_PATH
	originalEnv := os.Getenv("CHROME_PATH")
	defer os.Setenv("CHROME_PATH", originalEnv)

	os.Setenv("CHROME_PATH", "/env/chrome")

	// Empty explicit path should fall back to env
	result := ResolveChromePath("")
	if result != "/env/chrome" {
		t.Errorf("expected CHROME_PATH to be used, got %s", result)
	}

	// Explicit path should take precedence over env
	result = ResolveChromePath("/explicit/chrome")
	if result != "/explicit/chrome" {
		t.Errorf("expected explicit path to take precedence, got %s", result)
	}
}

func TestResolveChromePath_SystemDefault(t *testing.T) {
	// Clear CHROME_PATH
	originalEnv := os.Getenv("CHROME_PATH")
	defer os.Setenv("CHROME_PATH", originalEnv)
	os.Unsetenv("CHROME_PATH")

	// Empty explicit path and no env should fall back to system default
	result := ResolveChromePath("")

	// Result may be empty if no system Chrome is found, or a valid path
	// We just verify the function doesn't panic
	t.Logf("System default Chrome path: %s (empty is valid if Chrome not installed)", result)
}

func TestResolveExecutable(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPath bool // whether we expect a non-empty result
	}{
		{
			name:     "existing command",
			input:    "go", // go should exist in test environment
			wantPath: true,
		},
		{
			name:     "non-existing command",
			input:    "definitely-not-a-real-command-xyz123",
			wantPath: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveExecutable(tt.input)
			if tt.wantPath && result == "" {
				t.Errorf("expected path for %s, got empty", tt.input)
			}
			if !tt.wantPath && result != "" {
				t.Errorf("expected empty for %s, got %s", tt.input, result)
			}
		})
	}
}

func TestResolveChromePath_NotFound(t *testing.T) {
	// Clear CHROME_PATH
	originalEnv := os.Getenv("CHROME_PATH")
	defer os.Setenv("CHROME_PATH", originalEnv)
	os.Unsetenv("CHROME_PATH")

	// Clear PATH to ensure no Chrome is found
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)
	os.Setenv("PATH", "")

	// On a clean environment with no Chrome, should return empty
	// Note: This test may still find Chrome on macOS/Windows via absolute paths
	result := ResolveChromePath("")
	t.Logf("Result with empty PATH: %s (may still find Chrome via absolute path)", result)
}

func TestResolveExecutable_FullPath(t *testing.T) {
	// Test with a known existing path
	var testPath string
	switch runtime.GOOS {
	case "windows":
		testPath = os.Getenv("COMSPEC") // Usually C:\Windows\System32\cmd.exe
	default:
		testPath = "/bin/sh" // Should exist on all Unix-like systems
	}

	if testPath == "" {
		t.Skip("No known executable path for this platform")
	}

	result := resolveExecutable(testPath)
	if result != testPath {
		t.Errorf("expected %s, got %s", testPath, result)
	}

	// Test with non-existing path
	result = resolveExecutable("/definitely/not/a/real/path/chrome")
	if result != "" {
		t.Errorf("expected empty for non-existing path, got %s", result)
	}
}

// TestResolveChromePath_AbsoluteFallback covers the case the Playwright
// download used to be the only answer to: a browser that is installed, but
// not reachable through PATH.
//
// This is what happens in a container entrypoint, a cron job, or any process
// started with a trimmed environment — and it is worth resolving locally
// rather than by downloading 150 MB.
func TestResolveChromePath_AbsoluteFallback(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("absolute candidate list is Linux-specific")
	}

	// Find a browser the normal way first; if the machine has none, there is
	// nothing for the absolute paths to discover either.
	withPath := ResolveChromePath("")
	if withPath == "" {
		t.Skip("no browser installed on this machine")
	}

	originalPath := os.Getenv("PATH")
	originalChrome := os.Getenv("CHROME_PATH")
	t.Cleanup(func() {
		os.Setenv("PATH", originalPath)
		os.Setenv("CHROME_PATH", originalChrome)
	})
	os.Unsetenv("CHROME_PATH")
	os.Setenv("PATH", "/nonexistent")

	got, err := ResolveChromePathErr("")
	if got == "" {
		t.Fatalf("no browser found with PATH cleared, though %s exists: %v", withPath, err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected an absolute path with PATH cleared, got %q", got)
	}
}
