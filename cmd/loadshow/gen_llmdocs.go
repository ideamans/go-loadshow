package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/urfave/cli/v2"
)

// docsDir is where the embedded reference chapters live, relative to this
// package's directory (go generate runs in the directory of the directive).
const docsDir = "../../internal/llmdocs"

// localeFixed marks the re-executed child process. See pinLocale.
const localeFixed = "GEN_LLMDOCS_LOCALE_FIXED"

// pinLocale makes the generated catalog independent of the machine that
// produced it.
//
// Flag and command descriptions go through go-l10n, which resolves its
// language from LANG / LANGUAGE / LC_ALL / LC_MESSAGES while packages
// initialise — before main() gets control, so ForceLanguage would be too
// late. Without pinning, the catalog comes out in a different language
// depending on the environment and CI disagrees with whoever generated last.
//
// loadshow's primary README is English (README.ja.md is the translation), so
// the catalog is generated in English. go-l10n falls back to the source
// strings only when none of the locale variables are set — setting LANG=C is
// not enough, it still selects the Japanese lexicon — so they are stripped
// from the child environment rather than overridden.
func pinLocale() {
	if os.Getenv(localeFixed) != "" {
		return
	}
	child := exec.Command(os.Args[0], os.Args[1:]...)
	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		switch strings.SplitN(kv, "=", 2)[0] {
		case "LANG", "LANGUAGE", "LC_ALL", "LC_MESSAGES":
			continue
		}
		env = append(env, kv)
	}
	child.Env = append(env, localeFixed+"=1")
	child.Stdout, child.Stderr = os.Stdout, os.Stderr
	if err := child.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "gen-llmdocs:", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// generateCatalog walks the urfave/cli tree and writes the command catalog
// chapter.
//
// go-llm-cli-kit's catalog package only understands cobra, so this is the
// urfave equivalent. Output is sorted so the file is deterministic and a diff
// only ever shows a real change.
func generateCatalog(app *cli.App) error {
	var b strings.Builder
	b.WriteString("# Command catalog\n\n")
	b.WriteString("Generated from the urfave/cli command tree by `go generate ./...`.\n")
	b.WriteString("Do not edit by hand — edit the command definitions instead.\n")

	if flags := visibleFlags(app.Flags); len(flags) > 0 {
		b.WriteString("\n## Global flags\n\n")
		writeFlagTable(&b, flags)
	}

	cmds := append([]*cli.Command(nil), app.Commands...)
	sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })
	for _, c := range cmds {
		writeCommand(&b, app.Name, c)
	}

	path := filepath.Join(docsDir, "90-commands.md")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d bytes)\n", path, b.Len())
	return nil
}

func writeCommand(b *strings.Builder, prefix string, c *cli.Command) {
	// `llm` documents itself in 00-guide.md; gen-llmdocs is a development
	// command and must not be advertised to agents.
	if c.Hidden || c.Name == "llm" || c.Name == "gen-llmdocs" {
		return
	}
	full := prefix + " " + c.Name
	fmt.Fprintf(b, "\n## `%s`\n\n", full)
	if c.Usage != "" {
		fmt.Fprintf(b, "%s\n", c.Usage)
	}
	if c.ArgsUsage != "" {
		fmt.Fprintf(b, "\n```\n%s %s\n```\n", full, c.ArgsUsage)
	}
	if c.Description != "" {
		fmt.Fprintf(b, "\n%s\n", strings.TrimSpace(c.Description))
	}
	if flags := visibleFlags(c.Flags); len(flags) > 0 {
		b.WriteString("\n")
		writeFlagTable(b, flags)
	}
	subs := append([]*cli.Command(nil), c.Subcommands...)
	sort.Slice(subs, func(i, j int) bool { return subs[i].Name < subs[j].Name })
	for _, s := range subs {
		writeCommand(b, full, s)
	}
}

func visibleFlags(flags []cli.Flag) []cli.Flag {
	out := make([]cli.Flag, 0, len(flags))
	for _, f := range flags {
		if df, ok := f.(cli.DocGenerationFlag); ok && !isVisible(f) {
			_ = df
			continue
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Names()[0] < out[j].Names()[0] })
	return out
}

func isVisible(f cli.Flag) bool {
	type visibler interface{ IsVisible() bool }
	if v, ok := f.(visibler); ok {
		return v.IsVisible()
	}
	return true
}

func writeFlagTable(b *strings.Builder, flags []cli.Flag) {
	b.WriteString("| flag | default | description |\n| --- | --- | --- |\n")
	for _, f := range flags {
		names := make([]string, 0, len(f.Names()))
		for _, n := range f.Names() {
			if len(n) == 1 {
				names = append(names, "`-"+n+"`")
			} else {
				names = append(names, "`--"+n+"`")
			}
		}
		def, usage := "—", ""
		if df, ok := f.(cli.DocGenerationFlag); ok {
			if d := df.GetDefaultText(); d != "" {
				def = "`" + d + "`"
			}
			usage = df.GetUsage()
		}
		fmt.Fprintf(b, "| %s | %s | %s |\n",
			strings.Join(names, ", "), def, strings.ReplaceAll(usage, "|", "\\|"))
	}
}
