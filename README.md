# loadshow

[日本語版 README](README.ja.md)

A CLI tool that records web page loading as MP4 video for web performance visualization.

## Features

- Records web page loading process as scrolling video
- AV1 video encoding (via libaom) for high quality at small file sizes
- Desktop and mobile presets for quick configuration
- Network throttling (simulate slow connections)
- CPU throttling (simulate slower devices)
- Customizable layout, colors, and styling
- Cross-platform: Linux, macOS, Windows

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

## Usage

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

### Debug Mode

```bash
# Enable debug output (saves intermediate frames)
loadshow record https://example.com -o output.mp4 -d --debug-dir ./debug
```

## All Options

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

loadshow uses a pipeline architecture:

1. **Layout Stage** - Calculate video layout based on config
2. **Record Stage** - Capture screenshots during page load via Chrome DevTools Protocol
3. **Banner Stage** - Generate info banner with timing data
4. **Composite Stage** - Render screenshots into video frames
5. **Encode Stage** - Encode frames to AV1/MP4

## License

MIT License
