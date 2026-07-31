# loadshow — reference for AI agents

`loadshow` records a web page loading as an MP4 video, laid out as scrolling
columns so a whole page fits one frame. `juxtapose` puts two such videos
side by side for a before/after comparison. It exists to make web performance
visible to people who will not read a waterfall chart.

**The repository is `go-loadshow`; the command is `loadshow`.**

It drives Chrome, so a Chrome or Chromium install is required. Nothing prompts;
it reads flags only. Errors go to stderr and the exit code is non-zero on
failure. This reference is embedded in the binary, so `loadshow llm` always
describes the exact version you are running.

## Ground rules

1. **`--output` is required and the file is overwritten.** Both `record` and
   `juxtapose` need `-o`. Check the path before running.
2. **A recording is not a measurement.** The video shows one load under
   whatever conditions the machine had at that moment. Do not present it as a
   performance metric, and do not compare two recordings taken with different
   throttling, presets or network conditions.
3. **To compare fairly, pin the conditions on both runs.** Same `--preset`,
   same `--viewport-width`, same `--download-mbps` / `--upload-mbps`, same
   `--cpu-throttling`. `juxtapose` will happily combine two videos that are not
   comparable.
4. **Recording takes real time.** The page has to load, and `--timeout-sec`
   (default 30) bounds it. Encoding a long page at high quality is slower still
   — do not assume a hung process.

## Commands

| Task | Command |
| --- | --- |
| Record a page load | `loadshow record <url> -o out.mp4` |
| Compare two recordings | `loadshow juxtapose <left.mp4> <right.mp4> -o cmp.mp4` |

### record

```bash
loadshow record https://example.com -o before.mp4 \
  --preset mobile --download-mbps 10 --cpu-throttling 4
```

The defaults model a mid-range phone: `--preset mobile`, output 512×640.
`--preset desktop` switches the viewport and user agent.

Throttling is what makes a recording representative:

- `--download-mbps` / `--upload-mbps` — network speed, `0` means unlimited
- `--cpu-throttling` — slowdown factor, `4.0` is roughly a low-end phone
- `--viewport-width` — browser width (minimum 500), separate from output size

Layout: `--columns` splits a tall page into that many columns, with `--gap`,
`--margin`, `--indent` and `--outdent` controlling the arrangement.

Quality: `--quality low|medium|high` is the simple control. `--video-crf`
(0–63, lower is better) and `--screencast-quality` override it when you need
something specific. `--codec h264` (default) or `av1`.

`--output-summary out.md` writes a Markdown summary of the run — **use it when
reporting**, rather than describing the video from memory.

### juxtapose

```bash
loadshow juxtapose before.mp4 after.mp4 -o comparison.mp4 --gap 8
```

Both inputs should have been recorded with the same settings — see ground
rule 3.

## Failure modes

| Symptom | Cause | Fix |
| --- | --- | --- |
| Chrome not found | no Chrome/Chromium on the machine | install one, or pass `--chrome-path` |
| times out on a slow page | default 30 s | raise `--timeout-sec` |
| TLS certificate error | staging host with a self-signed certificate | `--ignore-https-errors` |
| H.264 encoding fails on Linux | Linux uses ffmpeg for H.264 | install ffmpeg, or pass `--ffmpeg-path`, or use `--codec av1` |
| the video looks nothing like a real visit | no throttling applied | set `--download-mbps` and `--cpu-throttling` |
| two videos are not comparable | different presets or throttling | re-record both with identical settings |
| output file missing | `-o` not passed | `--output` is required |

## What this CLI will not do

- It does not measure. No Lighthouse score, no Core Web Vitals, no waterfall.
  Use `crux` or Lighthouse for numbers; `loadshow` is for showing.
- It does not crawl or log in. One URL per `record`.
- It does not host or upload the video anywhere.
