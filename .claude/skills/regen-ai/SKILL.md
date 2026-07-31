---
name: regen-ai
description: Regenerate the embedded LLM reference for loadshow and verify it still builds and passes the plugin checks.
---

# regen-ai

```bash
go generate ./...
git diff --stat -- internal/llmdocs
go test ./cmd/loadshow
go run ./cmd/loadshow llm | head -5
```

The catalog must come out in English. If it turns Japanese, the locale
stripping in `cmd/loadshow/gen_llmdocs.go` has been removed or broken.
