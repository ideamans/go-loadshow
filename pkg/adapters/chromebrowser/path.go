// Package chromebrowser provides a browser implementation using chromedp.
package chromebrowser

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/playwright-community/playwright-go"
)

// ResolveChromePath resolves the Chrome executable path in the following order:
// 1. If explicitPath is non-empty, use it
// 2. If CHROME_PATH environment variable is set, use it
// 3. Fall back to system defaults (chromium → chrome order per platform)
// 4. If no system Chrome found, auto-install Chromium via Playwright
func ResolveChromePath(explicitPath string) string {
	path, _ := ResolveChromePathErr(explicitPath)
	return path
}

// ResolveChromePathErr is ResolveChromePath, but explains why it came up empty.
//
// The distinction matters: "no browser on this machine" and "the automatic
// install failed halfway" need different responses from the caller, and the
// second used to be reported as the first — the underlying error was
// discarded, so a CI failure said only "chrome not found".
func ResolveChromePathErr(explicitPath string) (string, error) {
	// 1. Explicit path from CLI
	if explicitPath != "" {
		return explicitPath, nil
	}

	// 2. CHROME_PATH environment variable
	if envPath := os.Getenv("CHROME_PATH"); envPath != "" {
		return envPath, nil
	}

	// 3. System defaults
	return findSystemChrome()
}

// findSystemChrome searches for Chrome/Chromium in system default locations.
// It tries Chromium first, then Chrome, to prefer the more lightweight browser.
// If no system Chrome is found, it falls back to auto-installing via Playwright.
func findSystemChrome() (string, error) {
	var candidates []string

	switch runtime.GOOS {
	case "darwin":
		// macOS: Try Chromium first, then Chrome
		candidates = []string{
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
		}
	case "linux":
		// Linux: Try chromium variants first, then chrome.
		//
		// Bare names are looked up on PATH; the absolute paths after them
		// cover the common case of a browser that is installed but not
		// reachable through PATH — a trimmed PATH, a cron job, a container
		// entrypoint. Without them the only remaining fallback is a network
		// download, which is a poor answer when the browser is already on
		// the disk.
		candidates = []string{
			"chromium",
			"chromium-browser",
			"google-chrome-stable",
			"google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/google-chrome",
			"/opt/google/chrome/chrome",
		}
	case "windows":
		// Windows: Try common installation paths
		programFiles := os.Getenv("PROGRAMFILES")
		programFilesX86 := os.Getenv("PROGRAMFILES(X86)")
		localAppData := os.Getenv("LOCALAPPDATA")

		if programFiles != "" {
			candidates = append(candidates,
				programFiles+"\\Chromium\\Application\\chrome.exe",
				programFiles+"\\Google\\Chrome\\Application\\chrome.exe",
			)
		}
		if programFilesX86 != "" {
			candidates = append(candidates,
				programFilesX86+"\\Chromium\\Application\\chrome.exe",
				programFilesX86+"\\Google\\Chrome\\Application\\chrome.exe",
			)
		}
		if localAppData != "" {
			candidates = append(candidates,
				localAppData+"\\Chromium\\Application\\chrome.exe",
				localAppData+"\\Google\\Chrome\\Application\\chrome.exe",
			)
		}
	}

	for _, candidate := range candidates {
		if path := resolveExecutable(candidate); path != "" {
			return path, nil
		}
	}

	// No system Chrome found, try to install via Playwright
	return installChromiumViaPlaywright()
}

// resolveExecutable checks if the given path/name exists as an executable.
// For full paths, it checks if the file exists.
// For command names, it uses exec.LookPath.
func resolveExecutable(nameOrPath string) string {
	// Check if it's a full path
	if len(nameOrPath) > 0 && (nameOrPath[0] == '/' || (len(nameOrPath) > 1 && nameOrPath[1] == ':')) {
		if _, err := os.Stat(nameOrPath); err == nil {
			return nameOrPath
		}
		return ""
	}

	// Try to find in PATH
	if path, err := exec.LookPath(nameOrPath); err == nil {
		return path
	}

	return ""
}

// installChromiumViaPlaywright installs Chromium using Playwright and returns the executable path.
// This is used as a fallback when no system Chrome/Chromium is found.
func installChromiumViaPlaywright() (string, error) {
	// Install Chromium browser via Playwright
	if err := playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium"},
	}); err != nil {
		return "", fmt.Errorf("playwright could not install chromium: %w", err)
	}

	// Get the Chromium executable path from Playwright's installation directory
	return getPlaywrightChromiumPath()
}

// getPlaywrightChromiumPath returns the path to Playwright-installed Chromium executable.
func getPlaywrightChromiumPath() (string, error) {
	// Playwright installs browsers in a cache directory
	// The path varies by OS but follows a pattern
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("no user cache directory: %w", err)
	}

	// Playwright stores browsers under ms-playwright directory
	playwrightDir := filepath.Join(cacheDir, "ms-playwright")

	// Find the chromium directory (version may vary)
	entries, err := os.ReadDir(playwrightDir)
	if err != nil {
		return "", fmt.Errorf("playwright reported success but %s is unreadable: %w", playwrightDir, err)
	}

	// Look for the chromium-<revision> directory.
	//
	// Match on the "chromium-" prefix, not the first eight characters:
	// Playwright also installs "chromium_headless_shell-<revision>" beside it,
	// which starts with the same eight characters but contains
	// chrome-linux/headless_shell rather than a full browser.
	var chromiumDir string
	var seen []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		seen = append(seen, entry.Name())
		if strings.HasPrefix(entry.Name(), "chromium-") {
			chromiumDir = filepath.Join(playwrightDir, entry.Name())
			break
		}
	}

	if chromiumDir == "" {
		return "", fmt.Errorf("no chromium-* directory under %s (found: %s)",
			playwrightDir, strings.Join(seen, ", "))
	}

	// Platform-specific executable path within the chromium directory
	var execPath string
	switch runtime.GOOS {
	case "darwin":
		execPath = filepath.Join(chromiumDir, "chrome-mac", "Chromium.app", "Contents", "MacOS", "Chromium")
	case "linux":
		execPath = filepath.Join(chromiumDir, "chrome-linux", "chrome")
	case "windows":
		execPath = filepath.Join(chromiumDir, "chrome-win", "chrome.exe")
	default:
		return "", errors.New("unsupported platform for playwright chromium: " + runtime.GOOS)
	}

	if _, err := os.Stat(execPath); err != nil {
		return "", fmt.Errorf("playwright chromium is installed but %s is missing: %w", execPath, err)
	}
	return execPath, nil
}
