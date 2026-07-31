package main

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ideamans/go-loadshow/internal/llmdocs"
)

// llmCommand prints the reference embedded in this binary.
//
// go-llm-cli-kit's llmcmd package only speaks cobra, so the subcommand is
// wired by hand here. The rendering itself still comes from the shared
// llmdocs package, so the output contract — Markdown by default, a JSON array
// of chapters under --format json — matches every other ideamans CLI.
func llmCommand() *cli.Command {
	return &cli.Command{
		Name:  "llm",
		Usage: "Print the reference for AI agents embedded in this binary",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "format",
				Value: "markdown",
				Usage: "output format: markdown or json",
			},
		},
		Action: func(c *cli.Context) error {
			docs := llmdocs.Docs()
			switch strings.ToLower(c.String("format")) {
			case "", "markdown", "md":
				out, err := docs.Markdown()
				if err != nil {
					return err
				}
				fmt.Fprint(c.App.Writer, out)
			case "json":
				out, err := docs.JSON()
				if err != nil {
					return err
				}
				fmt.Fprintln(c.App.Writer, string(out))
			default:
				return fmt.Errorf("unknown format %q: use markdown or json", c.String("format"))
			}
			return nil
		},
	}
}

// handleLegacyLLMFlag keeps the historical `--llm` spelling working from any
// position on the command line, including after a subcommand. urfave rejects
// an unknown flag on a leaf command, so this has to run before app.Run.
// Everything after `--` is an operand and is not scanned.
func handleLegacyLLMFlag(args []string) (bool, error) {
	format := ""
	for i, a := range args[1:] {
		if a == "--" {
			break
		}
		switch {
		case a == "--llm":
			format = "markdown"
			// `--llm json` and `--llm=json` are both accepted.
			if i+2 < len(args) && !strings.HasPrefix(args[i+2], "-") {
				format = args[i+2]
			}
		case strings.HasPrefix(a, "--llm="):
			format = strings.TrimPrefix(a, "--llm=")
		}
	}
	if format == "" {
		return false, nil
	}

	docs := llmdocs.Docs()
	switch strings.ToLower(format) {
	case "markdown", "md":
		out, err := docs.Markdown()
		if err != nil {
			return true, err
		}
		fmt.Print(out)
	case "json":
		out, err := docs.JSON()
		if err != nil {
			return true, err
		}
		fmt.Println(string(out))
	default:
		return true, fmt.Errorf("unknown format %q: use markdown or json", format)
	}
	return true, nil
}
