# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**loadshow** is a Go CLI tool and library that records web page loading as MP4/WebM video for web performance visualization. It captures page load via Chrome DevTools Protocol and renders the results as scrolling video with multi-column layout support.

Key characteristics:
- Single-binary distribution (no external dependencies on Windows/macOS)
- H.264 (OS native) and AV1 (libaom static-linked) codec support
- Hexagonal architecture with pipeline-based processing
- Cross-platform: Linux, macOS, Windows

## Development Commands

```bash
# Install dependencies (dynamic linking, for development)
make deps

# Build CLI binary
make build

# Run all tests (unit + integration)
make test

# Run unit tests only
make unit-test

# Run quick unit tests (no race detector)
make test-quick
# or directly:
go test ./pkg/...

# Run integration tests (requires Chrome)
make integration-test

# Run E2E tests (requires Chrome + network)
make e2e-test

# Lint code
make lint

# Format code
go fmt ./...

# Clean build artifacts
make clean
```

### Release Build

```bash
# Install static dependencies
make deps-static

# Build static binary with version
make build-static VERSION=v1.0.0

# Create release archive
make package VERSION=v1.0.0
```

## Architecture

### Pipeline Architecture

```
CLI (urfave/cli/v2)
    ↓
Orchestrator (coordinates stages)
    ↓
Pipeline Stages (sequential):
    1. Layout Stage     → Calculate video layout dimensions (pure function)
    2. Record Stage     → Capture screenshots via Chrome DevTools Protocol
    3. Banner Stage     → Generate info banner with timing data
    4. Composite Stage  → Render frames into video frames (parallel workers)
    5. Encode Stage     → Encode frames to MP4/WebM
    ↓
Output Video File
```

### Hexagonal Architecture (Ports & Adapters)

**Ports** (`pkg/ports/`) - Interface definitions:
- `Browser` - Web automation (chromedp)
- `VideoEncoder` - Video encoding (h264, av1)
- `VideoDecoder` - Video decoding
- `Renderer` - Image manipulation (fogleman/gg)
- `FileSystem` - File I/O
- `DebugSink` - Debug output
- `Logger` - Logging with i18n
- `HTMLCapturer` - HTML to image capture

**Adapters** (`pkg/adapters/`) - Implementations:
- `chromebrowser/` - Browser via chromedp
- `h264encoder/` - H.264 encoding (VideoToolbox/Media Foundation/FFmpeg)
- `av1encoder/` - AV1 encoding (libaom)
- `smartencoder/` - Auto-detection with fallback chain
- `ggrenderer/` - Image rendering via fogleman/gg
- `osfilesystem/` - OS file operations

### Key Package Structure

```
pkg/
├── ports/           # Interface definitions
├── adapters/        # Interface implementations
├── pipeline/        # Stage interface and types (Stage[In, Out])
├── stages/          # Pipeline stage implementations
│   ├── layout/      # Layout calculation (pure function, no deps)
│   ├── record/      # Browser recording
│   ├── banner/      # Banner generation
│   ├── composite/   # Frame composition (parallel)
│   └── encode/      # Video encoding
├── orchestrator/    # Stage coordination
├── juxtapose/       # Side-by-side video comparison
├── loadshow/        # High-level API with ConfigBuilder
├── config/          # YAML configuration
└── mocks/           # Test mocks for all ports

cmd/
└── loadshow/        # CLI entry point
```

## Testing Patterns

- **Unit tests**: Use mock implementations from `pkg/mocks/`
- **Layout stage**: Pure function testing (no mocks needed)
- **Table-driven tests**: Standard Go testing approach
- **Integration tests**: Require Chrome browser
- **E2E tests**: Require Chrome + network access

Run a single test:
```bash
go test -v -run TestLayoutStage ./pkg/stages/layout/
```

## Codec Support

**H.264** (default):
- macOS: VideoToolbox (native API)
- Windows: Media Foundation (native API)
- Linux: FFmpeg (external dependency)

**AV1**:
- All platforms: libaom (static-linked)

**Fallback chain**: Native H.264 → FFmpeg → AV1 (with warning)

## CLI Commands

```bash
# Record page load
loadshow record <url> -o <output.mp4>

# Side-by-side comparison
loadshow juxtapose <left.mp4> <right.mp4> -o <comparison.mp4>

# Show version
loadshow version
```

### Common Flags

- `-p, --preset`: `desktop` or `mobile`
- `-q, --quality`: `low`, `medium`, `high`
- `--codec`: `h264` or `av1`
- `-d, --debug`: Enable debug output
- `-c, --columns`: Number of columns for layout

## Build Requirements

- Go 1.24+
- Chrome/Chromium browser (runtime)
- libaom (for AV1 encoding)
- FFmpeg (Linux only, for H.264)
