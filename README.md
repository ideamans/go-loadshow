# loadshow

[日本語版 README](README.ja.md)

A CLI tool and Go library that records web page loading as MP4 video for web performance visualization.

## Features

- Records web page loading process as scrolling video
- AV1 video encoding (via libaom) for high quality at small file sizes
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

- Chrome or Chromium browser (automatically detected, or set `CHROME_PATH`)

## CLI Usage

### Commands

```
loadshow record <url> -o <output>     Record a web page loading as MP4 video
loadshow juxtapose <left> <right> -o <output>  Create a side-by-side comparison video
loadshow version                       Show version information
```

### Basic Recording

```bash
# Record a page with desktop preset
loadshow record https://example.com -o output.mp4

# Record with mobile preset
loadshow record https://example.com -o output.mp4 -p mobile
```

### Video Options

```bash
# Custom video dimensions
loadshow record https://example.com -o output.mp4 -W 640 -H 480

# Higher quality (lower CRF = better quality, larger file)
loadshow record https://example.com -o output.mp4 -q 20
```

### Network Throttling

```bash
# Simulate slow 3G connection (50KB/s download)
loadshow record https://example.com -o output.mp4 --download-speed 51200
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
```

### Debug Mode

```bash
# Enable debug output (saves intermediate frames)
loadshow record https://example.com -o output.mp4 -d --debug-dir ./debug
```

## All Options

### record

```
Usage: loadshow record <url> -o <output> [flags]

Arguments:
  <url>    URL of the page to record

Flags:
  -o, --output=STRING          Output MP4 file path (required)
  -p, --preset="desktop"       Preset: desktop or mobile
  -W, --width=INT              Output video width
  -H, --height=INT             Output video height
      --viewport-width=INT     Browser viewport width
  -c, --columns=INT            Number of columns
      --margin=INT             Margin around canvas
      --gap=INT                Gap between columns
      --indent=INT             Top margin for columns 2+
      --outdent=INT            Bottom margin for column 1
      --background-color=STR   Background color (hex)
      --border-color=STR       Border color (hex)
      --border-width=INT       Border width in pixels
  -q, --quality=INT            Video quality (CRF 0-63)
      --outro-ms=INT           Final frame hold duration (ms)
      --credit=STRING          Custom banner text
      --download-speed=INT     Download throttle (bytes/sec)
      --upload-speed=INT       Upload throttle (bytes/sec)
      --cpu-throttling=FLOAT   CPU slowdown factor
  -d, --debug                  Enable debug output
      --debug-dir=STRING       Debug output directory
      --no-headless            Run browser visibly
      --chrome-path=STRING     Path to Chrome
      --ignore-https-errors    Ignore cert errors
      --proxy-server=STRING    HTTP proxy server
      --no-incognito           Disable incognito mode
  -l, --log-level="info"       Log level: debug,info,warn,error
  -Q, --quiet                  Suppress all log output
```

### juxtapose

```
Usage: loadshow juxtapose <left> <right> -o <output>

Arguments:
  <left>   Left video file path
  <right>  Right video file path

Flags:
  -o, --output=STRING    Output MP4 file path (required)
```

## Go Library Usage

loadshow can also be used as a Go library for programmatic video generation.

### Installation

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

    "github.com/user/loadshow/pkg/adapters/av1encoder"
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
    // Create configuration with desktop preset
    cfg := loadshow.NewConfigBuilder().
        WithWidth(512).
        WithHeight(640).
        WithColumns(3).
        WithQuality(30).
        Build()

    // Or use mobile preset
    // cfg := loadshow.NewMobileConfigBuilder().Build()

    // Create adapters
    fs := osfilesystem.New()
    renderer := ggrenderer.New()
    browser := chromebrowser.New()
    htmlCapturer := capturehtml.New()
    encoder := av1encoder.New()
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
builder.WithQuality(30)          // CRF 0-63 (lower = better)
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

    "github.com/user/loadshow/pkg/adapters/av1decoder"
    "github.com/user/loadshow/pkg/adapters/av1encoder"
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
    decoder := av1decoder.NewMP4Reader()
    defer decoder.Close()

    encoder := av1encoder.New()
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

```
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

```
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
│   ├── av1encoder/  # AV1 video encoding
│   ├── av1decoder/  # AV1 video decoding
│   ├── chromebrowser/
│   ├── ggrenderer/
│   └── ...
├── juxtapose/       # Side-by-side video comparison
└── mocks/           # Test mocks
```

## License

MIT License
