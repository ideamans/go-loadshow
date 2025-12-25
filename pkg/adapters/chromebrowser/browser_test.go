package chromebrowser

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/user/loadshow/pkg/ports"
)

func TestBrowser_Launch_AutoInstallChromium(t *testing.T) {
	// This test verifies that when Chrome is not found in system paths,
	// it gets automatically installed via Playwright and launch succeeds.
	// This test only works on Linux where Chrome paths are searched via PATH
	if runtime.GOOS != "linux" {
		t.Skip("Auto-install test only reliable on Linux")
	}

	// Save and clear environment
	originalEnv := os.Getenv("CHROME_PATH")
	originalPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("CHROME_PATH", originalEnv)
		os.Setenv("PATH", originalPath)
	}()

	os.Unsetenv("CHROME_PATH")
	os.Setenv("PATH", "/nonexistent") // Set PATH to non-existent directory

	browser := New()
	ctx := context.Background()

	// No explicit path, no CHROME_PATH, no Chrome in PATH
	// Playwright should auto-install Chromium
	err := browser.Launch(ctx, ports.BrowserOptions{
		Headless: true,
	})

	if err != nil {
		t.Fatalf("expected auto-install to succeed, got error: %v", err)
	}
	defer browser.Close()
}

func TestBrowser_Launch_WithExplicitPath(t *testing.T) {
	// If Chrome is installed, test that explicit path works
	chromePath := ResolveChromePath("")
	if chromePath == "" {
		t.Skip("Chrome not installed, skipping explicit path test")
	}

	browser := New()
	ctx := context.Background()

	err := browser.Launch(ctx, ports.BrowserOptions{
		ChromePath: chromePath,
		Headless:   true,
	})

	if err != nil {
		t.Fatalf("failed to launch with explicit path: %v", err)
	}
	defer browser.Close()
}
