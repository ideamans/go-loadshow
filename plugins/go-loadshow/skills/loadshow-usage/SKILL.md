---
name: loadshow-usage
description: Record a web page loading as an MP4 video and put two recordings side by side, using the loadshow CLI, with network and CPU throttling so the result resembles a real visit. Use when the user asks to show or visualise how slowly a page loads, wants a before/after video of a performance improvement, needs something to show a client or a non-technical stakeholder about page speed, or asks to record a page load.
license: MIT
compatibility: Requires the `loadshow` binary on PATH — run the loadshow-install skill if it is missing — and a Chrome or Chromium install for it to drive. On Linux, H.264 output additionally needs ffmpeg; AV1 does not.
allowed-tools: Bash(loadshow:*) Bash(command:*) Bash(ls:*) Read
---

# loadshow-usage

Page-load videos, and honest comparisons between them.

## 1. Confirm the tool

```bash
command -v loadshow && loadshow --version
```

Missing binary? Run the `loadshow-install` skill. It also needs Chrome or
Chromium; on Linux, H.264 encoding needs ffmpeg — `--codec av1` avoids that
dependency.

## 2. A recording is not a measurement

This is the mistake that matters. The video shows **one load, on this machine,
at this moment**. It is a demonstration, not a metric.

- Never report a duration from a recording as the page's load time.
- Never compare two recordings taken with different settings.
- If the user wants numbers, say so and point at `crux` or Lighthouse. If they
  want something to *show* someone, this is the right tool.

## 3. Throttle, or the video is misleading

An unthrottled recording on a developer laptop makes almost any page look fast.
Set conditions explicitly and say what you used:

```bash
loadshow record https://example.com -o before.mp4 \
  --preset mobile --download-mbps 10 --cpu-throttling 4
```

`--preset` is `mobile` (default) or `desktop`. `--cpu-throttling 4` is roughly
a low-end phone. `--download-mbps` / `--upload-mbps` take `0` for unlimited.

## 4. For a before/after, pin every setting

```bash
loadshow record "$URL_BEFORE" -o before.mp4 --preset mobile --download-mbps 10 --cpu-throttling 4
loadshow record "$URL_AFTER"  -o after.mp4  --preset mobile --download-mbps 10 --cpu-throttling 4
loadshow juxtapose before.mp4 after.mp4 -o comparison.mp4 --gap 8
```

Same preset, same viewport, same throttling, both times. `juxtapose` will
combine two videos regardless — it cannot tell you they were not comparable,
so that check is yours.

Recording takes real time: the page has to load, bounded by `--timeout-sec`
(default 30). A long encode is not a hang.

## 5. Read the reference for anything else

```bash
loadshow llm
loadshow record --help
```

## 6. Report from the summary, not from memory

```bash
loadshow record "$URL" -o out.mp4 --output-summary out.md
```

`--output-summary` writes a Markdown summary of the run. Quote that, state the
throttling you applied, and give the output path. **Say that the video is a
demonstration under stated conditions** — not a benchmark.

## Failure modes

| Symptom | Fix |
| --- | --- |
| `command not found: loadshow` | run the `loadshow-install` skill |
| Chrome not found | install Chrome/Chromium, or pass `--chrome-path` |
| timeout on a slow page | raise `--timeout-sec` |
| TLS error on staging | `--ignore-https-errors` |
| H.264 fails on Linux | install ffmpeg, pass `--ffmpeg-path`, or use `--codec av1` |
| page looks unrealistically fast | no throttling — set `--download-mbps` and `--cpu-throttling` |
| no output file | `--output` / `-o` is required |
| looked for `go-loadshow` | the command is `loadshow`; only the repository carries the `go-` prefix |
