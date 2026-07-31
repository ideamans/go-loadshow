---
paths:
  - "cmd/loadshow/**/*.go"
  - "internal/llmdocs/0*.md"
  - "plugins/go-loadshow/**"
  - "context7.json"
---

# Regen triggers

Before committing:

1. Run `/regen-ai` (or `go generate ./...`) and commit the result.
2. Do not hand-edit `internal/llmdocs/90-commands.md`.
3. If you changed a skill description, `go test ./cmd/loadshow` —
   `TestPluginSkills` asserts the discovery keywords are still present.
