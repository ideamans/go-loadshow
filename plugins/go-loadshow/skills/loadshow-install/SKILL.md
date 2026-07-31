---
name: loadshow-install
description: Make the loadshow command available, installing it only if it is missing. Use when another skill reports that `loadshow` is not on PATH, or when the user asks to install, update or upgrade the ideamans page-load video CLI. Prefers an already-installed binary, then the latest GitHub release, then a build from source.
license: MIT
compatibility: Requires curl and tar to install from a release. Standalone — does not need loadshow to be present already. Installs from the public repository github.com/ideamans/go-loadshow, so no GitHub authentication is needed. Building from source needs a Go toolchain and cgo (the video encoders are native). The tool needs Chrome or Chromium at run time, which this skill does not install.
allowed-tools: Bash(curl:*) Bash(wget:*) Bash(tar:*) Bash(unzip:*) Bash(go:*) Bash(uname:*) Bash(command:*) Bash(which:*) Bash(mkdir:*) Bash(mv:*) Bash(cp:*) Bash(rm:*) Bash(chmod:*) Bash(ls:*) Bash(test:*) Bash(echo:*) Read
---

# loadshow-install

Make the `loadshow` command usable, doing the least work that achieves it.

The command is `loadshow`. The repository is `go-loadshow`.

## Route 1 — an existing installation on PATH

```bash
command -v loadshow && loadshow --version
```

If that resolves, **use it and stop here.** Do not check for a newer release —
it costs an API call and the user did not ask for an upgrade.

Two checks before trusting the hit:

- **It is the right tool.** `loadshow llm | head -1` must read
  `# loadshow — reference for AI agents`. If something else owns the name, say
  so and use an explicit path rather than shadowing theirs.
- **It is recent enough.** If `llm` is not a known command, the binary predates
  the embedded reference — continue to route 2 to upgrade it.

## Route 2 — the latest GitHub release

The repository is public, so no authentication is needed.

```bash
VERSION=$(curl -fsSL https://api.github.com/repos/ideamans/go-loadshow/releases/latest \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)   # e.g. v1.6.0
```

Asset names keep the **`v` prefix** and use lowercase OS and arch:

```
loadshow_<version-with-v>_<os>_<arch>.tar.gz
```

`<os>` is `darwin`, `linux` or `windows`; `<arch>` is `amd64` or `arm64`, so
`uname -m` reporting `x86_64` maps to `amd64`. Windows ships a `.zip`.
**There is no `darwin_amd64` build** — Intel Macs fall through to route 3.

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]')             # darwin | linux
ARCH=$(uname -m); [ "$ARCH" = "x86_64" ] && ARCH=amd64  # amd64 | arm64
curl -fsSL -o /tmp/loadshow.tar.gz \
  "https://github.com/ideamans/go-loadshow/releases/download/${VERSION}/loadshow_${VERSION}_${OS}_${ARCH}.tar.gz"
```

If the download 404s, list the release's actual assets rather than retrying
variations.

### Install onto PATH

```bash
tar -xzf /tmp/loadshow.tar.gz -C /tmp
mkdir -p ~/.local/bin && mv /tmp/loadshow ~/.local/bin/ && chmod +x ~/.local/bin/loadshow
```

Prefer the first writable directory already on PATH — `~/.local/bin`, then
`/usr/local/bin`. Two things not to do on your own initiative:

- If nothing on PATH is writable, leave the binary in `/tmp`, print the exact
  `sudo mv` command and let the user run it. Do not run `sudo` yourself.
- If `~/.local/bin` is not on PATH, give the user the line for their shell
  profile. Do not edit the profile for them.

## Route 3 — build from source

```bash
go install github.com/ideamans/go-loadshow/cmd/loadshow@latest
```

**This needs cgo and native video encoding libraries**, so it is slower and
more fragile than the other two routes — the release binaries exist precisely
to avoid it. If the build fails on a missing library, prefer route 2 or tell
the user which library is missing rather than trying to install it yourself.

## Verify

```bash
command -v loadshow && loadshow --version && loadshow llm | head -1
```

Report the version and the path. Then say what is still needed: **Chrome or
Chromium must be installed** for recording, and on Linux, H.264 output also
needs ffmpeg (`--codec av1` avoids that). This skill installs neither.
