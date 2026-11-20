package chromebrowser

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/ideamans/go-loadshow/pkg/ports"
)

// TestBrowser_Launch_AutoInstallChromium exercises the last-resort path:
// download Chromium through Playwright when the machine has no browser.
//
// It is opt-in, because it is not a unit test — it pulls a driver and a
// ~150 MB browser from a third-party CDN. That dependency has broken once:
// playwright-go v0.5200.1 fetched its driver from *.azureedge.net, which was
// retired. Upstream now publishes as github.com/mxschmitt/playwright-go and
// downloads from the npm registry and the Playwright CDN. Where the browser
// lands inside the cache is covered offline by TestFindPlaywrightChromium.
//
// Set LOADSHOW_TEST_PLAYWRIGHT_INSTALL=1 to run it.
func TestBrowser_Launch_AutoInstallChromium(t *testing.T) {
	if os.Getenv("LOADSHOW_TEST_PLAYWRIGHT_INSTALL") == "" {
		t.Skip("set LOADSHOW_TEST_PLAYWRIGHT_INSTALL=1 to exercise the Playwright download path")
	}
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
