# AI artifact policy

`loadshow llm` prints an embedded reference assembled from
`internal/llmdocs/`. One chapter is generated; the rest are hand-written.

## Never hand-edit

| File | Produced by |
| --- | --- |
| `internal/llmdocs/90-commands.md` | `go generate ./...` → the hidden `gen-llmdocs` mode in `cmd/loadshow` |

## Source of truth

| To change… | Edit |
| --- | --- |
| ground rules, throttling guidance, failure modes | `internal/llmdocs/00-guide.md` |
| a command or flag description | the urfave/cli definition in `cmd/loadshow/main.go` |
| what the distributed skills tell an agent | `plugins/go-loadshow/skills/*/SKILL.md` |
| pitfalls surfaced through context7 | `context7.json` `rules` |

## Two things specific to this repository

**The catalog generator is hand-written.** `go-llm-cli-kit`'s `catalog`
package only understands cobra; `cmd/loadshow/gen_llmdocs.go` is the urfave
equivalent. Adding a command or flag needs no change there, but changing the
CLI framework does.

**The generator strips locale variables before walking the tree.** Descriptions
go through go-l10n, which selects the Japanese lexicon whenever any of
`LANG` / `LANGUAGE` / `LC_ALL` / `LC_MESSAGES` is set — including `LANG=C`.
Only an environment with none of them set yields the English source strings.
Remove that stripping and the committed catalog starts flipping language
depending on who ran `go generate`.
