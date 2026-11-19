package chromebrowser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// touch creates an empty file (and its parents) under root.
func touch(t *testing.T, root string, rel string) string {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFindPlaywrightChromium(t *testing.T) {
	tests := []struct {
		name         string
		goos, goarch string
		files        []string
		want         string // relative to the ms-playwright dir; "" means an error
	}{
		{
			name: "mac arm64, Chrome for Testing layout",
			goos: "darwin", goarch: "arm64",
			files: []string{"chromium-1254/chrome-mac-arm64/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing"},
			want:  "chromium-1254/chrome-mac-arm64/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing",
		},
		{
			name: "mac, legacy Chromium.app layout still found",
			goos: "darwin", goarch: "arm64",
			files: []string{"chromium-1169/chrome-mac/Chromium.app/Contents/MacOS/Chromium"},
			want:  "chromium-1169/chrome-mac/Chromium.app/Contents/MacOS/Chromium",
		},
		{
			name: "linux amd64",
			goos: "linux", goarch: "amd64",
			files: []string{"chromium-1254/chrome-linux64/chrome"},
			want:  "chromium-1254/chrome-linux64/chrome",
		},
		{
			name: "linux arm64",
			goos: "linux", goarch: "arm64",
			files: []string{"chromium-1254/chrome-linux-arm64/chrome"},
			want:  "chromium-1254/chrome-linux-arm64/chrome",
		},
		{
			name: "windows",
			goos: "windows", goarch: "amd64",
			files: []string{"chromium-1254/chrome-win64/chrome.exe"},
			want:  "chromium-1254/chrome-win64/chrome.exe",
		},
		{
			name: "newest revision wins regardless of directory order",
			goos: "linux", goarch: "amd64",
			files: []string{
				"chromium-999/chrome-linux/chrome",
				"chromium-1254/chrome-linux64/chrome",
				"chromium-1100/chrome-linux/chrome",
			},
			want: "chromium-1254/chrome-linux64/chrome",
		},
		{
			name: "headless shell is not mistaken for the browser",
			goos: "linux", goarch: "amd64",
			files: []string{"chromium_headless_shell-1254/chrome-headless-shell-linux64/chrome-headless-shell"},
			want:  "",
		},
		{
			name: "falls back to an older revision when the newest is incomplete",
			goos: "linux", goarch: "amd64",
			files: []string{
				"chromium-1254/INSTALLATION_COMPLETE",
				"chromium-1200/chrome-linux64/chrome",
			},
			want: "chromium-1200/chrome-linux64/chrome",
		},
		{
			name: "unsupported platform",
			goos: "plan9", goarch: "amd64",
			files: []string{"chromium-1254/chrome-linux64/chrome"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				touch(t, dir, f)
			}
			got, err := findPlaywrightChromium(dir, tt.goos, tt.goarch)
			if tt.want == "" {
				if err == nil {
					t.Fatalf("expected an error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if want := filepath.Join(dir, filepath.FromSlash(tt.want)); got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func TestFindPlaywrightChromium_MissingDir(t *testing.T) {
	_, err := findPlaywrightChromium(filepath.Join(t.TempDir(), "nope"), "linux", "amd64")
	if err == nil || !strings.Contains(err.Error(), "unreadable") {
		t.Fatalf("expected an unreadable-directory error, got %v", err)
	}
}
