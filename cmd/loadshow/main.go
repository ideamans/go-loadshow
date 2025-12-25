// Package main provides the CLI entry point for loadshow.
package main

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/ideamans/go-l10n"

	"github.com/user/loadshow/pkg/adapters/av1encoder"
	"github.com/user/loadshow/pkg/adapters/capturehtml"
	"github.com/user/loadshow/pkg/adapters/chromebrowser"
	"github.com/user/loadshow/pkg/adapters/filesink"
	"github.com/user/loadshow/pkg/adapters/ggrenderer"
	"github.com/user/loadshow/pkg/adapters/logger"
	"github.com/user/loadshow/pkg/adapters/nullsink"
	"github.com/user/loadshow/pkg/adapters/osfilesystem"
	"github.com/user/loadshow/pkg/config"
	"github.com/user/loadshow/pkg/loadshow"
	"github.com/user/loadshow/pkg/orchestrator"
	"github.com/user/loadshow/pkg/ports"
	"github.com/user/loadshow/pkg/stages/banner"
	"github.com/user/loadshow/pkg/stages/composite"
	"github.com/user/loadshow/pkg/stages/encode"
	"github.com/user/loadshow/pkg/stages/layout"
	"github.com/user/loadshow/pkg/stages/record"
)

// CLI defines the command-line interface with subcommands.
type CLI struct {
	Record    RecordCmd    `cmd:"" help:"Record a web page loading as MP4 video."`
	Juxtapose JuxtaposeCmd `cmd:"" help:"Create a side-by-side comparison video."`
	Version   VersionCmd   `cmd:"" help:"Show version information."`
}

// RecordCmd defines the record subcommand.
type RecordCmd struct {
	// Required arguments
	URL    string `arg:"" help:"URL of the page to record."`
	Output string `short:"o" required:"" help:"Output MP4 file path."`

	// Preset
	Preset string `short:"p" default:"desktop" enum:"desktop,mobile" help:"Preset configuration (desktop or mobile)."`

	// Video dimensions
	Width  *int `short:"W" help:"Output video width (default: 512)."`
	Height *int `short:"H" help:"Output video height (default: 640)."`

	// Layout options (override preset)
	ViewportWidth *int `help:"Browser viewport width (min: 500)."`
	Columns       *int `short:"c" help:"Number of columns (min: 1)."`
	Margin        *int `help:"Margin around the canvas in pixels."`
	Gap           *int `help:"Gap between columns in pixels."`
	Indent        *int `help:"Additional top margin for columns 2+."`
	Outdent       *int `help:"Additional bottom margin for column 1."`

	// Style options
	BackgroundColor *string `help:"Background color (hex, e.g., #dcdcdc)."`
	BorderColor     *string `help:"Border color (hex, e.g., #b4b4b4)."`
	BorderWidth     *int    `help:"Border width in pixels."`

	// Encoding options
	Quality *int `short:"q" help:"Video quality (CRF 0-63, lower is better)."`
	OutroMs *int `help:"Duration to hold final frame in milliseconds."`

	// Banner options
	Credit *string `help:"Custom text shown in banner (default: loadshow)."`

	// Network throttling
	DownloadSpeed *int `help:"Download speed in bytes/sec (0 = unlimited)."`
	UploadSpeed   *int `help:"Upload speed in bytes/sec (0 = unlimited)."`

	// CPU throttling
	CPUThrottling *float64 `help:"CPU slowdown factor (1.0 = no throttling, 4.0 = 4x slower)."`

	// Debug options
	Debug    bool   `short:"d" help:"Enable debug output."`
	DebugDir string `default:"./debug" help:"Directory for debug output."`

	// Browser options
	NoHeadless        bool   `help:"Run browser in non-headless mode."`
	ChromePath        string `help:"Path to Chrome executable (falls back to CHROME_PATH env, then system default)."`
	IgnoreHTTPSErrors bool   `help:"Ignore HTTPS certificate errors."`
	ProxyServer       string `help:"HTTP proxy server (e.g., http://proxy:8080)."`
	NoIncognito       bool   `help:"Disable incognito mode (incognito is enabled by default)."`

	// Logging options
	LogLevel string `short:"l" default:"info" enum:"debug,info,warn,error" help:"Log level (debug, info, warn, error)."`
	Quiet    bool   `short:"Q" help:"Suppress all log output."`
}

// JuxtaposeCmd defines the juxtapose subcommand.
type JuxtaposeCmd struct {
	// TODO: Implement juxtapose command
	Left   string `arg:"" help:"Left video file path."`
	Right  string `arg:"" help:"Right video file path."`
	Output string `short:"o" required:"" help:"Output MP4 file path."`
}

// VersionCmd shows version information.
type VersionCmd struct{}

var version = "dev"

func main() {
	cli := CLI{}

	ctx := kong.Parse(&cli,
		kong.Name("loadshow"),
		kong.Description("Create page load videos for web performance visualization."),
		kong.UsageOnError(),
	)

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

// Run executes the record command.
func (cmd *RecordCmd) Run() error {
	// Build config from preset and overrides
	cfg := cmd.buildConfig()

	// Create logger
	var log ports.Logger
	if cmd.Quiet {
		log = logger.NewNoop()
	} else {
		log = logger.NewConsole(ports.ParseLogLevel(cmd.LogLevel))
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Warn(l10n.T("Interrupted, shutting down..."))
		cancel()
	}()

	// Create adapters
	fs := osfilesystem.New()
	renderer := ggrenderer.New()
	browser := chromebrowser.New()
	htmlCapturer := capturehtml.New()

	// Create AV1 encoder
	encoder := av1encoder.New()

	// Create debug sink
	var sink ports.DebugSink
	if cmd.Debug {
		if err := fs.MkdirAll(cmd.DebugDir); err != nil {
			return fmt.Errorf("create debug directory: %w", err)
		}
		sink = filesink.New(cmd.DebugDir, fs, renderer)
	} else {
		sink = nullsink.New()
	}

	// Determine number of workers
	workers := runtime.NumCPU()

	// Create stages
	layoutStage := layout.NewStage()
	recordStage := record.New(browser, sink, log, ports.BrowserOptions{
		Headless:          !cmd.NoHeadless,
		ChromePath:        cmd.ChromePath,
		Incognito:         !cmd.NoIncognito,
		IgnoreHTTPSErrors: cmd.IgnoreHTTPSErrors,
		ProxyServer:       cmd.ProxyServer,
	})
	bannerStage := banner.NewStage(htmlCapturer, sink, log)
	compositeStage := composite.NewStage(renderer, sink, log, workers)
	encodeStage := encode.NewStage(encoder, log)

	// Create orchestrator
	orch := orchestrator.New(
		layoutStage,
		recordStage,
		bannerStage,
		compositeStage,
		encodeStage,
		fs,
		sink,
		log,
	)

	// Build orchestrator config
	orchConfig := cfg.ToOrchestratorConfig(cmd.URL, cmd.Output)

	// Print start message
	log.Info(l10n.F("Recording %s (%s preset)...", cmd.URL, cmd.Preset))

	// Run pipeline
	if err := orch.Run(ctx, orchConfig); err != nil {
		return err
	}

	log.Info(l10n.F("Output saved to %s", cmd.Output))
	return nil
}

// buildConfig creates a Config from preset and CLI overrides.
func (cmd *RecordCmd) buildConfig() loadshow.Config {
	// Start with preset
	var builder *loadshow.ConfigBuilder
	switch cmd.Preset {
	case "mobile":
		builder = loadshow.NewMobileConfigBuilder()
	default:
		builder = loadshow.NewConfigBuilder()
	}

	// Apply video dimensions
	if cmd.Width != nil {
		builder.WithWidth(*cmd.Width)
	}
	if cmd.Height != nil {
		builder.WithHeight(*cmd.Height)
	}

	// Apply overrides
	if cmd.ViewportWidth != nil {
		builder.WithViewportWidth(*cmd.ViewportWidth)
	}
	if cmd.Columns != nil {
		builder.WithColumns(*cmd.Columns)
	}
	if cmd.Margin != nil {
		builder.WithMargin(*cmd.Margin)
	}
	if cmd.Gap != nil {
		builder.WithGap(*cmd.Gap)
	}
	if cmd.Indent != nil {
		builder.WithIndent(*cmd.Indent)
	}
	if cmd.Outdent != nil {
		builder.WithOutdent(*cmd.Outdent)
	}
	if cmd.BackgroundColor != nil {
		builder.WithBackgroundColor(config.ParseColor(*cmd.BackgroundColor))
	}
	if cmd.BorderColor != nil {
		builder.WithBorderColor(config.ParseColor(*cmd.BorderColor))
	}
	if cmd.BorderWidth != nil {
		builder.WithBorderWidth(*cmd.BorderWidth)
	}
	if cmd.Quality != nil {
		builder.WithQuality(*cmd.Quality)
	}
	if cmd.OutroMs != nil {
		builder.WithOutroMs(*cmd.OutroMs)
	}
	if cmd.Credit != nil {
		builder.WithCredit(*cmd.Credit)
	}
	if cmd.DownloadSpeed != nil {
		builder.WithDownloadSpeed(*cmd.DownloadSpeed)
	}
	if cmd.UploadSpeed != nil {
		builder.WithUploadSpeed(*cmd.UploadSpeed)
	}
	if cmd.CPUThrottling != nil {
		builder.WithCPUThrottling(*cmd.CPUThrottling)
	}

	// Apply browser options
	if cmd.IgnoreHTTPSErrors {
		builder.WithIgnoreHTTPSErrors(true)
	}
	if cmd.ProxyServer != "" {
		builder.WithProxyServer(cmd.ProxyServer)
	}

	return builder.Build()
}

// Run executes the juxtapose command.
func (cmd *JuxtaposeCmd) Run() error {
	fmt.Println(l10n.T("Juxtapose command not yet implemented."))
	fmt.Println(l10n.F("Would create comparison from %s and %s to %s", cmd.Left, cmd.Right, cmd.Output))
	return nil
}

// Run executes the version command.
func (cmd *VersionCmd) Run() error {
	fmt.Println(l10n.F("loadshow (Go) version %s", version))
	return nil
}

// parseHexColor parses a hex color string to color.Color.
func parseHexColor(hex string) color.Color {
	return config.ParseColor(hex)
}
