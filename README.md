# loadshow

[日本語版 README](README.ja.md)

A CLI tool and Go library that records web page loading as MP4 video for web performance visualization.

## Features

- Records web page loading process as scrolling video
- H.264 video encoding by default (uses OS native APIs on Windows/macOS)
- AV1 video encoding available for high quality at small file sizes
- **Single binary distribution**: No external dependencies on Windows and macOS
- Desktop and mobile presets for quick configuration
- Network throttling (simulate slow connections)
- CPU throttling (simulate slower devices)
- Juxtapose command to create side-by-side comparison videos
- Customizable layout, colors, and styling
- Cross-platform: Linux, macOS, Windows
- Usable as both CLI tool and Go library

## Installation

### Download Binary

Download the latest release from [GitHub Releases](https://github.com/user/loadshow/releases).

```bash
# Linux (amd64)
curl -LO https://github.com/user/loadshow/releases/latest/download/loadshow_vX.X.X_linux_amd64.tar.gz
tar -xzf loadshow_vX.X.X_linux_amd64.tar.gz
sudo mv loadshow /usr/local/bin/

# macOS (arm64)
curl -LO https://github.com/user/loadshow/releases/latest/download/loadshow_vX.X.X_darwin_arm64.tar.gz
tar -xzf loadshow_vX.X.X_darwin_arm64.tar.gz
sudo mv loadshow /usr/local/bin/
```

### Build from Source

Requires Go 1.21+ and libaom.

```bash
# Install dependencies
make deps

# Build
make build
```

## Requirements

### All Platforms

- Chrome or Chromium browser (automatically detected via Playwright, or set `CHROME_PATH`)

### Platform-Specific Dependencies

| Platform    | H.264 Codec                    | AV1 Codec              | External Dependencies      |
|-------------|--------------------------------|------------------------|----------------------------|
| **Windows** | Media Foundation (OS built-in) | libaom (static linked) | None                       |
| **macOS**   | VideoToolbox (OS built-in)     | libaom (static linked) | None                       |
| **Linux**   | FFmpeg (external)              | libaom (static linked) | FFmpeg required for H.264  |

**H.264 Encoder Fallback Chain:**
When `--codec h264` is specified, loadshow tries encoders in this order:
1. **Native encoder** (VideoToolbox on macOS, Media Foundation on Windows)
2. **FFmpeg** (if native is unavailable)
3. **AV1 fallback** (if both native and FFmpeg are unavailable, with warning)

On Linux, install FFmpeg for H.264 support:

```bash
# Ubuntu/Debian
sudo apt-get install ffmpeg

# Or use AV1 codec (no external dependencies)
loadshow record https://example.com -o output.mp4 --codec av1
```

## CLI Usage

### Commands

```text
loadshow record <url> -o <output>     Record a web page loading as MP4 video
loadshow juxtapose <left> <right> -o <output>  Create a side-by-side comparison video
loadshow version                       Show version information
```

### Basic Recording

```bash
# Record a page with mobile preset (default)
loadshow record https://example.com -o output.mp4

# Record with desktop preset
loadshow record https://example.com -o output.mp4 -p desktop
```

### Quality Presets

```bash
# Low quality (fast, smaller file)
loadshow record https://example.com -o output.mp4 -q low

# High quality (slower, larger file)
loadshow record https://example.com -o output.mp4 -q high

# Custom video CRF (0-63, lower = better quality, overrides preset)
loadshow record https://example.com -o output.mp4 --video-crf 20

# Custom screencast quality (0-100, overrides preset)
loadshow record https://example.com -o output.mp4 --screencast-quality 90
```

### Video Codec Options

```bash
# Use H.264 codec (default, best compatibility)
loadshow record https://example.com -o output.mp4 --codec h264

# Use AV1 codec (smaller file size, better quality)
loadshow record https://example.com -o output.mp4 --codec av1

# Specify FFmpeg path (Linux only, for H.264)
loadshow record https://example.com -o output.mp4 --ffmpeg-path /usr/bin/ffmpeg
```

### Video Dimensions

```bash
# Custom video dimensions
loadshow record https://example.com -o output.mp4 -W 640 -H 480
```

### Network Throttling

```bash
# Simulate slow connection (1.5 Mbps download)
loadshow record https://example.com -o output.mp4 --download-mbps 1.5

# Simulate slow upload (0.5 Mbps)
loadshow record https://example.com -o output.mp4 --upload-mbps 0.5
```

### CPU Throttling

```bash
# Simulate 4x slower CPU
loadshow record https://example.com -o output.mp4 --cpu-throttling 4.0
```

### Layout Customization

```bash
# Custom columns and gaps
loadshow record https://example.com -o output.mp4 -c 3 --gap 10 --margin 20

# Custom colors
loadshow record https://example.com -o output.mp4 --background-color "#f0f0f0" --border-color "#cccccc"
```

### Browser Options

```bash
# Use specific Chrome path
loadshow record https://example.com -o output.mp4 --chrome-path /path/to/chrome

# Run in visible mode (non-headless)
loadshow record https://example.com -o output.mp4 --no-headless

# Ignore HTTPS errors (self-signed certs)
loadshow record https://example.com -o output.mp4 --ignore-https-errors

# Use proxy
loadshow record https://example.com -o output.mp4 --proxy-server http://proxy:8080
```

### Juxtapose (Side-by-Side Comparison)

```bash
# Create a side-by-side comparison of two videos
loadshow juxtapose before.mp4 after.mp4 -o comparison.mp4

# Specify output codec (input codec is auto-detected)
loadshow juxtapose before.mp4 after.mp4 -o comparison.mp4 --codec av1
```

**Note:** The input video codec is automatically detected from the MP4 files. The `--codec` option only affects the output encoding. Both input videos must use the same codec (either both H.264 or both AV1).

### Debug Mode

```bash
# Enable debug output (saves intermediate frames)
loadshow record https://example.com -o output.mp4 -d --debug-dir ./debug
```

## All Options

### record

```text
Usage: loadshow record <url> -o <output> [flags]

Arguments:
  <url>    URL of the page to record

Flags:
  Output:
    -o, --output STRING        Output MP4 file path (required)

  Preset:
    -p, --preset STRING        Device preset: desktop, mobile (default: mobile)
    -q, --quality STRING       Quality preset: low, medium, high (default: medium)

  Browser:
        --viewport-width INT   Browser viewport width (min: 500)
        --chrome-path STRING   Path to Chrome executable
        --no-headless          Run browser in non-headless mode
        --no-incognito         Disable incognito mode
        --ignore-https-errors  Ignore HTTPS certificate errors
        --proxy-server STRING  HTTP proxy server (e.g., http://proxy:8080)

  Performance Emulation:
        --download-mbps FLOAT  Download speed in Mbps (0 = unlimited)
        --upload-mbps FLOAT    Upload speed in Mbps (0 = unlimited)
        --cpu-throttling FLOAT CPU slowdown factor (1.0 = no throttling)

  Layout and Style:
    -c, --columns INT          Number of columns (min: 1)
        --margin INT           Margin around canvas in pixels
        --gap INT              Gap between columns in pixels
        --indent INT           Additional top margin for columns 2+
        --outdent INT          Additional bottom margin for column 1
        --background-color STR Background color (hex, e.g., #dcdcdc)
        --border-color STR     Border color (hex, e.g., #b4b4b4)
        --border-width INT     Border width in pixels

  Banner:
        --credit STRING        Custom text shown in banner

  Video and Quality:
    -W, --width INT            Output video width
    -H, --height INT           Output video height
        --codec STRING         Video codec: h264, av1 (default: h264)
        --ffmpeg-path STRING   Path to FFmpeg executable (Linux H.264 only)
        --video-crf INT        Video CRF (0-63, overrides quality preset)
        --screencast-quality INT  Screencast JPEG quality (0-100, overrides preset)
        --outro-ms INT         Duration to hold final frame (ms)

  Debug:
    -d, --debug                Enable debug output
        --debug-dir STRING     Directory for debug output (default: ./debug)

  Logging:
    -l, --log-level STRING     Log level: debug, info, warn, error (default: info)
    -Q, --quiet                Suppress all log output
```

### juxtapose

```text
Usage: loadshow juxtapose <left> <right> -o <output> [flags]

Arguments:
  <left>   Left video file path
  <right>  Right video file path

Flags:
  Output:
    -o, --output STRING    Output MP4 file path (required)

  Preset:
    -q, --quality STRING   Quality preset: low, medium, high (default: medium)

  Layout and Style:
        --gap INT          Gap between videos in pixels (default: 10)

  Video and Quality:
        --codec STRING     Video codec: h264, av1 (default: h264)
        --ffmpeg-path STR  Path to FFmpeg executable (Linux H.264 only)
        --video-crf INT    Video CRF (0-63, overrides quality preset)
```

## Go Library Usage

loadshow can also be used as a Go library for programmatic video generation.

### Install Package

```bash
go get github.com/user/loadshow
```

### Basic Usage with ConfigBuilder

```go
package main

import (
    "context"
    "log"
    "runtime"

    "github.com/user/loadshow/pkg/adapters/h264encoder"  // or av1encoder for AV1
    "github.com/user/loadshow/pkg/adapters/chromebrowser"
    "github.com/user/loadshow/pkg/adapters/capturehtml"
    "github.com/user/loadshow/pkg/adapters/filesink"
    "github.com/user/loadshow/pkg/adapters/ggrenderer"
    "github.com/user/loadshow/pkg/adapters/logger"
    "github.com/user/loadshow/pkg/adapters/nullsink"
    "github.com/user/loadshow/pkg/adapters/osfilesystem"
    "github.com/user/loadshow/pkg/loadshow"
    "github.com/user/loadshow/pkg/orchestrator"
    "github.com/user/loadshow/pkg/ports"
    "github.com/user/loadshow/pkg/stages/banner"
    "github.com/user/loadshow/pkg/stages/composite"
    "github.com/user/loadshow/pkg/stages/encode"
    "github.com/user/loadshow/pkg/stages/layout"
    "github.com/user/loadshow/pkg/stages/record"
)

func main() {
    // Create configuration with mobile preset (default)
    cfg := loadshow.NewConfigBuilder().
        WithWidth(512).
        WithHeight(640).
        WithColumns(3).
        WithVideoCRF(30).
        Build()

    // Or use desktop preset
    // cfg := loadshow.NewDesktopConfigBuilder().Build()

    // Create adapters
    fs := osfilesystem.New()
    renderer := ggrenderer.New()
    browser := chromebrowser.New()
    htmlCapturer := capturehtml.New()
    encoder := h264encoder.New()  // Uses OS native API (Windows/macOS) or FFmpeg (Linux)
    sink := nullsink.New()
    log := logger.NewConsole(ports.LogLevelInfo)

    // Create pipeline stages
    layoutStage := layout.NewStage()
    recordStage := record.New(browser, sink, log, ports.BrowserOptions{
        Headless:  true,
        Incognito: true,
    })
    bannerStage := banner.NewStage(htmlCapturer, sink, log)
    compositeStage := composite.NewStage(renderer, sink, log, runtime.NumCPU())
    encodeStage := encode.NewStage(encoder, log)

    // Create and run orchestrator
    orch := orchestrator.New(
        layoutStage,
        recordStage,
        bannerStage,
        compositeStage,
        encodeStage,
        fs,
        sink,
        log,
    )

    orchConfig := cfg.ToOrchestratorConfig("https://example.com", "output.mp4")
    if err := orch.Run(context.Background(), orchConfig); err != nil {
        log.Fatal(err)
    }
}
```

### ConfigBuilder Methods

```go
// Video dimensions
builder.WithWidth(512)           // Output video width
builder.WithHeight(640)          // Output video height

// Layout options
builder.WithViewportWidth(375)   // Browser viewport width (min: 500)
builder.WithColumns(3)           // Number of columns (min: 1)
builder.WithMargin(20)           // Margin around canvas
builder.WithGap(20)              // Gap between columns
builder.WithIndent(20)           // Top margin for columns 2+
builder.WithOutdent(20)          // Bottom margin for column 1

// Style options
builder.WithBackgroundColor(color.RGBA{220, 220, 220, 255})
builder.WithBorderColor(color.RGBA{180, 180, 180, 255})
builder.WithBorderWidth(1)

// Encoding options
builder.WithVideoCRF(30)         // Video CRF 0-63 (lower = better)
builder.WithScreencastQuality(80) // Screencast JPEG quality 0-100
builder.WithOutroMs(2000)        // Final frame hold duration

// Network throttling
builder.WithDownloadSpeed(loadshow.Mbps(10))  // 10 Mbps
builder.WithUploadSpeed(loadshow.Mbps(5))     // 5 Mbps
builder.WithNetworkSpeed(loadshow.Mbps(10))   // Both directions

// CPU throttling
builder.WithCPUThrottling(4.0)   // 4x slower

// Browser options
builder.WithIgnoreHTTPSErrors(true)
builder.WithProxyServer("http://proxy:8080")

// Banner
builder.WithCredit("My Company")
```

### Juxtapose API

```go
package main

import (
    "context"
    "log"

    "github.com/user/loadshow/pkg/adapters/h264decoder"  // or av1decoder for AV1
    "github.com/user/loadshow/pkg/adapters/h264encoder"  // or av1encoder for AV1
    "github.com/user/loadshow/pkg/adapters/logger"
    "github.com/user/loadshow/pkg/adapters/osfilesystem"
    "github.com/user/loadshow/pkg/juxtapose"
)

func main() {
    // Simple function call
    err := juxtapose.Combine(
        "before.mp4",
        "after.mp4",
        "comparison.mp4",
        juxtapose.DefaultOptions(),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Or use Stage API for more control
    decoder := h264decoder.NewMP4Reader()  // Uses OS native API or FFmpeg
    defer decoder.Close()

    encoder := h264encoder.New()  // Uses OS native API or FFmpeg
    fs := osfilesystem.New()
    log := logger.NewConsole(ports.LogLevelInfo)

    opts := juxtapose.Options{
        Gap:     10,      // Gap between videos
        FPS:     30.0,    // Output frame rate
        Quality: 30,      // CRF quality
        Bitrate: 0,       // Auto bitrate
    }

    stage := juxtapose.New(decoder, encoder, fs, log, opts)
    result, err := stage.Execute(context.Background(), juxtapose.Input{
        LeftPath:   "before.mp4",
        RightPath:  "after.mp4",
        OutputPath: "comparison.mp4",
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Created %d frames, duration: %dms", result.FrameCount, result.DurationMs)
}
```

## Development

```bash
# Install dependencies (dynamic linking, for development)
make deps

# Build
make build

# Run tests
make test

# Run all tests including E2E
make test-all

# See all available targets
make help
```

### Release Build

```bash
# Install dependencies (static linking)
make deps-static

# Build static binary with version
make build-static VERSION=v1.0.0

# Create release archive
make package VERSION=v1.0.0
```

## Architecture

loadshow uses a pipeline architecture with dependency injection:

```text
┌─────────────────────────────────────────────────────────────┐
│                      Orchestrator                           │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────────┐│
│  │  Layout  │→ │  Record  │→ │  Banner  │→ │   Composite   ││
│  │  Stage   │  │  Stage   │  │  Stage   │  │     Stage     ││
│  └──────────┘  └──────────┘  └──────────┘  └───────────────┘│
│                                                      ↓      │
│                                             ┌───────────────┐│
│                                             │    Encode     ││
│                                             │     Stage     ││
│                                             └───────────────┘│
└─────────────────────────────────────────────────────────────┘
```

1. **Layout Stage** - Calculate video layout based on config
2. **Record Stage** - Capture screenshots during page load via Chrome DevTools Protocol
3. **Banner Stage** - Generate info banner with timing data
4. **Composite Stage** - Render screenshots into video frames
5. **Encode Stage** - Encode frames to AV1/MP4

### Package Structure

```text
pkg/
├── loadshow/        # High-level API with ConfigBuilder
├── orchestrator/    # Pipeline coordination
├── pipeline/        # Stage interfaces and types
├── stages/          # Pipeline stage implementations
│   ├── layout/      # Layout calculation
│   ├── record/      # Page recording
│   ├── banner/      # Banner generation
│   ├── composite/   # Frame composition
│   └── encode/      # Video encoding
├── ports/           # Interface definitions (ports)
├── adapters/        # Interface implementations (adapters)
│   ├── av1encoder/  # AV1 video encoding (libaom, static linked)
│   ├── av1decoder/  # AV1 video decoding (libaom, static linked)
│   ├── h264encoder/ # H.264 encoding (OS native or FFmpeg fallback)
│   ├── h264decoder/ # H.264 decoding (OS native or FFmpeg)
│   ├── codecdetect/ # Auto-detect video codec from MP4 files
│   ├── chromebrowser/
│   ├── ggrenderer/
│   └── ...
├── juxtapose/       # Side-by-side video comparison
└── mocks/           # Test mocks
```

## License

MIT License
