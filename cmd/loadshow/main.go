// Package main provides the CLI entry point for loadshow.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/alecthomas/kong"

	"github.com/user/loadshow/pkg/adapters/av1encoder"
	"github.com/user/loadshow/pkg/adapters/capturehtml"
	"github.com/user/loadshow/pkg/adapters/chromebrowser"
	"github.com/user/loadshow/pkg/adapters/filesink"
	"github.com/user/loadshow/pkg/adapters/ggrenderer"
	"github.com/user/loadshow/pkg/adapters/nullsink"
	"github.com/user/loadshow/pkg/adapters/osfilesystem"
	"github.com/user/loadshow/pkg/config"
	"github.com/user/loadshow/pkg/orchestrator"
	"github.com/user/loadshow/pkg/ports"
	"github.com/user/loadshow/pkg/stages/banner"
	"github.com/user/loadshow/pkg/stages/composite"
	"github.com/user/loadshow/pkg/stages/encode"
	"github.com/user/loadshow/pkg/stages/layout"
	"github.com/user/loadshow/pkg/stages/record"
)

// CLI defines the command-line interface.
type CLI struct {
	// Required arguments
	URL    string `arg:"" help:"URL of the page to record."`
	Output string `short:"o" required:"" help:"Output MP4 file path."`

	// Layout options
	CanvasWidth    int `short:"W" default:"512" help:"Canvas width in pixels."`
	CanvasHeight   int `short:"H" default:"640" help:"Canvas height in pixels."`
	Columns        int `short:"c" default:"3" help:"Number of columns."`
	Gap            int `default:"20" help:"Gap between columns in pixels."`
	Padding        int `default:"20" help:"Padding around the canvas in pixels."`
	BorderWidth    int `default:"1" help:"Border width for column frames."`
	Indent         int `default:"20" help:"Indent for non-first columns."`
	Outdent        int `default:"20" help:"Outdent for first column."`
	ProgressHeight int `default:"16" help:"Height of the progress bar."`

	// Recording options
	ViewportWidth int     `default:"1200" help:"Browser viewport width."`
	TimeoutMs     int     `default:"30000" help:"Recording timeout in milliseconds."`
	CPUThrottling float64 `default:"1.0" help:"CPU throttling rate (1.0 = no throttling)."`
	LatencyMs     int     `default:"0" help:"Network latency in milliseconds."`
	DownloadSpeed int     `default:"0" help:"Download speed in bytes/sec (0 = unlimited)."`
	UploadSpeed   int     `default:"0" help:"Upload speed in bytes/sec (0 = unlimited)."`
	UserAgent     string  `help:"Custom user agent string."`
	NoHeadless    bool    `help:"Run browser in non-headless mode."`
	ChromePath    string  `help:"Path to Chrome executable."`

	// Banner options
	Banner       bool `short:"b" help:"Enable banner generation."`
	BannerHeight int  `default:"80" help:"Banner height in pixels."`
	NoProgress   bool `help:"Disable progress bar."`

	// Encoding options
	Quality int     `short:"q" default:"20" help:"Video quality (CRF 0-63, lower is better)."`
	Bitrate int     `default:"0" help:"Target bitrate in kbps (0 = auto)."`
	FPS     float64 `default:"30.0" help:"Output video frame rate."`
	OutroMs int     `default:"2000" help:"Duration to hold final frame in milliseconds."`

	// Parallel processing
	Workers int `short:"w" default:"0" help:"Number of worker threads (0 = auto)."`

	// Debug options
	Debug    bool   `short:"d" help:"Enable debug output."`
	DebugDir string `default:"./debug" help:"Directory for debug output."`

	// Config file
	Config string `short:"C" type:"existingfile" help:"Configuration file (YAML)."`

	// Version
	Version kong.VersionFlag `short:"v" help:"Show version information."`
}

var version = "dev"

func main() {
	cli := CLI{}

	parser := kong.Must(&cli,
		kong.Name("loadshow"),
		kong.Description("Record web page loading as MP4 video with AV1 codec."),
		kong.UsageOnError(),
		kong.Vars{"version": version},
	)

	ctx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)

	// Handle errors
	if err := run(cli); err != nil {
		ctx.Errorf("%v", err)
		os.Exit(1)
	}
}

func run(cli CLI) error {
	// Load configuration
	cfg := buildConfig(cli)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nInterrupted, shutting down...")
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
	if cfg.Debug {
		if err := fs.MkdirAll(cfg.DebugDir); err != nil {
			return fmt.Errorf("create debug directory: %w", err)
		}
		sink = filesink.New(cfg.DebugDir, fs, renderer)
	} else {
		sink = nullsink.New()
	}

	// Determine number of workers
	workers := cfg.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Create stages
	layoutStage := layout.NewStage()
	recordStage := record.New(browser, sink, ports.BrowserOptions{
		Headless:   cfg.Headless,
		ChromePath: cfg.ChromePath,
		UserAgent:  cfg.UserAgent,
		Headers:    cfg.Headers,
	})
	bannerStage := banner.NewStage(htmlCapturer, sink)
	compositeStage := composite.NewStage(renderer, sink, workers)
	encodeStage := encode.NewStage(encoder)

	// Create orchestrator
	orch := orchestrator.New(
		layoutStage,
		recordStage,
		bannerStage,
		compositeStage,
		encodeStage,
		fs,
		sink,
	)

	// Build orchestrator config
	orchConfig := cfg.ToOrchestratorConfig()

	// Print start message
	fmt.Printf("Recording %s...\n", cfg.URL)

	// Run pipeline
	if err := orch.Run(ctx, orchConfig); err != nil {
		return err
	}

	fmt.Printf("Output saved to %s\n", cfg.OutputPath)
	return nil
}

func buildConfig(cli CLI) config.Config {
	cfg := config.Defaults()

	// Load from config file if specified
	if cli.Config != "" {
		if fileCfg, err := config.LoadFromFile(cli.Config); err == nil {
			cfg = fileCfg
		}
	}

	// Override with CLI arguments
	cfg.URL = cli.URL
	cfg.OutputPath = cli.Output

	// Layout
	cfg.CanvasWidth = cli.CanvasWidth
	cfg.CanvasHeight = cli.CanvasHeight
	cfg.Columns = cli.Columns
	cfg.Gap = cli.Gap
	cfg.Padding = cli.Padding
	cfg.BorderWidth = cli.BorderWidth
	cfg.Indent = cli.Indent
	cfg.Outdent = cli.Outdent
	cfg.ProgressHeight = cli.ProgressHeight

	// Recording
	cfg.ViewportWidth = cli.ViewportWidth
	cfg.TimeoutMs = cli.TimeoutMs
	cfg.CPUThrottling = cli.CPUThrottling
	cfg.Network.LatencyMs = cli.LatencyMs
	cfg.Network.DownloadSpeed = cli.DownloadSpeed
	cfg.Network.UploadSpeed = cli.UploadSpeed
	cfg.UserAgent = cli.UserAgent
	cfg.Headless = !cli.NoHeadless
	cfg.ChromePath = cli.ChromePath

	// Banner
	cfg.BannerEnabled = cli.Banner
	cfg.BannerHeight = cli.BannerHeight
	cfg.ShowProgress = !cli.NoProgress

	// Encoding
	cfg.Quality = cli.Quality
	cfg.Bitrate = cli.Bitrate
	cfg.FPS = cli.FPS
	cfg.OutroMs = cli.OutroMs

	// Workers
	cfg.Workers = cli.Workers

	// Debug
	cfg.Debug = cli.Debug
	cfg.DebugDir = cli.DebugDir

	return cfg
}
