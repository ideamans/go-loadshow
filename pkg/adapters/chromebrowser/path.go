// Package chromebrowser provides a browser implementation using chromedp.
package chromebrowser

import (
	"os"
	"os/exec"
	"runtime"
)

// ResolveChromePath resolves the Chrome executable path in the following order:
// 1. If explicitPath is non-empty, use it
// 2. If CHROME_PATH environment variable is set, use it
// 3. Fall back to system defaults (chromium → chrome order per platform)
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

	// Return empty string if no Chrome found; chromedp will use its own lookup
	return ""
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
