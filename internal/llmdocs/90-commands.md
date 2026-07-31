# Command catalog

Generated from the urfave/cli command tree by `go generate ./...`.
Do not edit by hand — edit the command definitions instead.

## `loadshow juxtapose`

Create a side-by-side comparison video

```
loadshow juxtapose <left> <right>
```

| flag | default | description |
| --- | --- | --- |
| `--border-color` | — | Border color between videos (hex, e.g., #505050) |
| `--codec` | `"h264"` | Video codec (h264, av1) |
| `--ffmpeg-path` | — | Path to ffmpeg executable (Linux only, for H.264) |
| `--gap` | `1` | Gap between videos in pixels |
| `--output`, `-o` | — | Output MP4 file path (required) |
| `--quality`, `-q` | `"medium"` | Quality preset (low, medium, high) |
| `--video-crf` | `0` | Video CRF value (0-63, lower is better, overrides quality preset) |

## `loadshow record`

Record a web page loading as MP4 video

```
loadshow record <url>
```

| flag | default | description |
| --- | --- | --- |
| `--background-color` | — | Background color (hex, e.g., #dcdcdc) |
| `--border-color` | — | Border color (hex, e.g., #b4b4b4) |
| `--border-width` | `0` | Border width in pixels |
| `--chrome-path` | — | Path to Chrome executable |
| `--codec` | `"h264"` | Video codec (h264, av1) |
| `--columns`, `-c` | `0` | Number of columns (min: 1) |
| `--cpu-throttling` | `0` | CPU slowdown factor (1.0 = no throttling, 4.0 = 4x slower) |
| `--credit` | — | Custom text shown in banner (default: loadshow) |
| `--debug`, `-d` | `false` | Enable debug output |
| `--debug-dir` | `"./debug"` | Directory for debug output |
| `--download-mbps` | `0` | Download speed in Mbps (0 = unlimited) |
| `--ffmpeg-path` | — | Path to ffmpeg executable (Linux only, for H.264) |
| `--gap` | `0` | Gap between columns in pixels |
| `--height`, `-H` | `0` | Output video height (default: 640) |
| `--ignore-https-errors` | `false` | Ignore HTTPS certificate errors |
| `--indent` | `0` | Additional top margin for columns 2+ |
| `--log-level`, `-l` | `"info"` | Log level (debug, info, warn, error) |
| `--margin` | `0` | Margin around the canvas in pixels |
| `--no-headless` | `false` | Run browser in non-headless mode |
| `--no-incognito` | `false` | Disable incognito mode |
| `--outdent` | `0` | Additional bottom margin for column 1 |
| `--output`, `-o` | — | Output MP4 file path (required) |
| `--output-summary` | — | Output execution summary to file (Markdown format) |
| `--outro-ms` | `0` | Duration to hold final frame in milliseconds |
| `--preset`, `-p` | `"mobile"` | Device preset (desktop, mobile) |
| `--proxy-server` | — | HTTP proxy server (e.g., http://proxy:8080) |
| `--quality`, `-q` | `"medium"` | Quality preset (low, medium, high) |
| `--quiet` | `false` | Suppress all log output |
| `--screencast-quality` | `0` | Screencast JPEG quality (0-100, overrides quality preset) |
| `--timeout-sec` | `30` | Recording timeout in seconds |
| `--upload-mbps` | `0` | Upload speed in Mbps (0 = unlimited) |
| `--video-crf` | `0` | Video CRF value (0-63, lower is better, overrides quality preset) |
| `--viewport-width` | `0` | Browser viewport width (min: 500) |
| `--width`, `-W` | `0` | Output video width (default: 512) |
