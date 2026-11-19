// Package chromebrowser provides a browser implementation using chromedp.
package chromebrowser

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/mxschmitt/playwright-go"
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
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("no user cache directory: %w", err)
	}

	// Playwright stores browsers under ms-playwright directory
	return findPlaywrightChromium(filepath.Join(cacheDir, "ms-playwright"), runtime.GOOS, runtime.GOARCH)
}

// findPlaywrightChromium looks for a Chromium executable under playwrightDir.
//
// Each install lives in a chromium-<revision> directory, and several revisions
// can sit side by side when the machine has run more than one Playwright
// release. The newest revision whose executable exists wins.
//
// Match on the "chromium-" prefix, not the first eight characters: Playwright
// also installs "chromium_headless_shell-<revision>" beside it, which starts
// with the same eight characters but holds a headless shell, not a browser.
func findPlaywrightChromium(playwrightDir, goos, goarch string) (string, error) {
	candidates := playwrightChromiumExecutables(goos, goarch)
	if len(candidates) == 0 {
		return "", errors.New("unsupported platform for playwright chromium: " + goos + "/" + goarch)
	}

	entries, err := os.ReadDir(playwrightDir)
	if err != nil {
		return "", fmt.Errorf("playwright reported success but %s is unreadable: %w", playwrightDir, err)
	}

	type install struct {
		dir      string
		revision int
	}
	var installs []install
	var seen []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		seen = append(seen, entry.Name())
		rev, ok := strings.CutPrefix(entry.Name(), "chromium-")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(rev)
		if err != nil {
			n = -1
		}
		installs = append(installs, install{dir: filepath.Join(playwrightDir, entry.Name()), revision: n})
	}

	if len(installs) == 0 {
		return "", fmt.Errorf("no chromium-* directory under %s (found: %s)",
			playwrightDir, strings.Join(seen, ", "))
	}

	sort.SliceStable(installs, func(i, j int) bool { return installs[i].revision > installs[j].revision })

	var tried []string
	for _, in := range installs {
		for _, rel := range candidates {
			execPath := filepath.Join(in.dir, rel)
			if _, err := os.Stat(execPath); err == nil {
				return execPath, nil
			}
			tried = append(tried, execPath)
		}
	}
	return "", fmt.Errorf("playwright chromium is installed but no executable was found (tried: %s)",
		strings.Join(tried, ", "))
}

// playwrightChromiumExecutables lists where the executable sits inside a
// chromium-<revision> directory, newest layout first.
//
// Newer Playwright releases ship Chromium as the Chrome for Testing build, which
// moved the executable (for example chrome-mac/Chromium.app became
// chrome-mac-arm64/Google Chrome for Testing.app). Older installs that are
// already on disk keep the old layout, so both are accepted.
func playwrightChromiumExecutables(goos, goarch string) []string {
	const cftApp = "Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing"
	switch goos {
	case "darwin":
		arch := "x64"
		if goarch == "arm64" {
			arch = "arm64"
		}
		return []string{
			filepath.FromSlash("chrome-mac-" + arch + "/" + cftApp),
			filepath.FromSlash("chrome-mac/Chromium.app/Contents/MacOS/Chromium"),
		}
	case "linux":
		dir := "chrome-linux64"
		if goarch == "arm64" {
			dir = "chrome-linux-arm64"
		}
		return []string{
			filepath.Join(dir, "chrome"),
			filepath.Join("chrome-linux", "chrome"),
		}
	case "windows":
		return []string{
			filepath.Join("chrome-win64", "chrome.exe"),
			filepath.Join("chrome-win", "chrome.exe"),
		}
	}
	return nil
}
