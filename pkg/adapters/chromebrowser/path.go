// Package chromebrowser provides a browser implementation using chromedp.
package chromebrowser

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/playwright-community/playwright-go"
)

// ResolveChromePath resolves the Chrome executable path in the following order:
// 1. If explicitPath is non-empty, use it
// 2. If CHROME_PATH environment variable is set, use it
// 3. Fall back to system defaults (chromium → chrome order per platform)
// 4. If no system Chrome found, auto-install Chromium via Playwright
func ResolveChromePath(explicitPath string) string {
	// 1. Explicit path from CLI
	if explicitPath != "" {
		return explicitPath
	}

	// 2. CHROME_PATH environment variable
	if envPath := os.Getenv("CHROME_PATH"); envPath != "" {
		return envPath
	}

	// 3. System defaults
	return findSystemChrome()
}

// findSystemChrome searches for Chrome/Chromium in system default locations.
// It tries Chromium first, then Chrome, to prefer the more lightweight browser.
// If no system Chrome is found, it falls back to auto-installing via Playwright.
func findSystemChrome() string {
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
		// Linux: Try chromium variants first, then chrome
		candidates = []string{
			"chromium",
			"chromium-browser",
			"google-chrome-stable",
			"google-chrome",
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
			return path
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
func installChromiumViaPlaywright() string {
	// Install Chromium browser via Playwright
	err := playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium"},
	})
	if err != nil {
		return ""
	}

	// Get the Chromium executable path from Playwright's installation directory
	return getPlaywrightChromiumPath()
}

// getPlaywrightChromiumPath returns the path to Playwright-installed Chromium executable.
func getPlaywrightChromiumPath() string {
	// Playwright installs browsers in a cache directory
	// The path varies by OS but follows a pattern
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}

	// Playwright stores browsers under ms-playwright directory
	playwrightDir := filepath.Join(cacheDir, "ms-playwright")

	// Find the chromium directory (version may vary)
	entries, err := os.ReadDir(playwrightDir)
	if err != nil {
		return ""
	}

	// Look for chromium-* directory
	var chromiumDir string
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) > 8 && entry.Name()[:8] == "chromium" {
			chromiumDir = filepath.Join(playwrightDir, entry.Name())
			break
		}
	}

	if chromiumDir == "" {
		return ""
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
		return ""
	}

	if _, err := os.Stat(execPath); err == nil {
		return execPath
	}

	return ""
}
