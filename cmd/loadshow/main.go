// Package main provides the CLI entry point for loadshow.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/ideamans/go-l10n"
	"github.com/spf13/cobra"

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

var version = "dev"

// Record command flags
var recordFlags struct {
	// Required
	output string

	// Preset
	preset  string
	quality string

	// Video output
	width    int
	height   int
	videoCRF int
	outroMs  int

	// Recording
	screencastQuality int
	viewportWidth     int

	// Layout
	columns int
	margin  int
	gap     int
	indent  int
	outdent int

	// Style
	backgroundColor string
	borderColor     string
	borderWidth     int

	// Network throttling
	downloadMbps float64
	uploadMbps   float64

	// CPU throttling
	cpuThrottling float64

	// Banner
	credit string

	// Browser
	noHeadless  bool
	chromePath  string
	ignoreHTTPS bool
	proxyServer string
	noIncognito bool

	// Debug
	debug    bool
	debugDir string

	// Logging
	logLevel string
	quiet    bool
}

// Juxtapose command flags
var juxtaposeFlags struct {
	output  string
	quality string
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "loadshow",
	Short: l10n.T("Create page load videos for web performance visualization"),
	Long:  l10n.T("loadshow creates videos that visualize web page loading performance."),
}

var recordCmd = &cobra.Command{
	Use:   "record <url>",
	Short: l10n.T("Record a web page loading as MP4 video"),
	Long:  l10n.T("Record the loading process of a web page and save it as an MP4 video."),
	Args:  cobra.ExactArgs(1),
	RunE:  runRecord,
}

var juxtaposeCmd = &cobra.Command{
	Use:   "juxtapose <left> <right>",
	Short: l10n.T("Create a side-by-side comparison video"),
	Long:  l10n.T("Create a side-by-side comparison video from two input videos."),
	Args:  cobra.ExactArgs(2),
	RunE:  runJuxtapose,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: l10n.T("Show version information"),
	Long:  l10n.T("Display the version of loadshow."),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(l10n.F("loadshow (Go) version %s", version))
	},
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(recordCmd)
	rootCmd.AddCommand(juxtaposeCmd)
	rootCmd.AddCommand(versionCmd)

	// ===== Required =====
	recordCmd.Flags().StringVarP(&recordFlags.output, "output", "o", "", l10n.T("Output MP4 file path (required)"))
	recordCmd.MarkFlagRequired("output")

	// ===== Preset =====
	recordCmd.Flags().StringVarP(&recordFlags.preset, "preset", "p", "mobile", l10n.T("Device preset (desktop, mobile)"))
	recordCmd.Flags().StringVarP(&recordFlags.quality, "quality", "q", "medium", l10n.T("Quality preset (low, medium, high)"))

	// ===== Video Output =====
	recordCmd.Flags().IntVarP(&recordFlags.width, "width", "W", 0, l10n.T("Output video width (default: 512)"))
	recordCmd.Flags().IntVarP(&recordFlags.height, "height", "H", 0, l10n.T("Output video height (default: 640)"))
	recordCmd.Flags().IntVar(&recordFlags.videoCRF, "video-crf", 0, l10n.T("Video CRF value (0-63, lower is better, overrides quality preset)"))
	recordCmd.Flags().IntVar(&recordFlags.outroMs, "outro-ms", 0, l10n.T("Duration to hold final frame in milliseconds"))

	// ===== Recording =====
	recordCmd.Flags().IntVar(&recordFlags.screencastQuality, "screencast-quality", 0, l10n.T("Screencast JPEG quality (0-100, overrides quality preset)"))
	recordCmd.Flags().IntVar(&recordFlags.viewportWidth, "viewport-width", 0, l10n.T("Browser viewport width (min: 500)"))

	// ===== Layout =====
	recordCmd.Flags().IntVarP(&recordFlags.columns, "columns", "c", 0, l10n.T("Number of columns (min: 1)"))
	recordCmd.Flags().IntVar(&recordFlags.margin, "margin", 0, l10n.T("Margin around the canvas in pixels"))
	recordCmd.Flags().IntVar(&recordFlags.gap, "gap", 0, l10n.T("Gap between columns in pixels"))
	recordCmd.Flags().IntVar(&recordFlags.indent, "indent", 0, l10n.T("Additional top margin for columns 2+"))
	recordCmd.Flags().IntVar(&recordFlags.outdent, "outdent", 0, l10n.T("Additional bottom margin for column 1"))

	// ===== Style =====
	recordCmd.Flags().StringVar(&recordFlags.backgroundColor, "background-color", "", l10n.T("Background color (hex, e.g., #dcdcdc)"))
	recordCmd.Flags().StringVar(&recordFlags.borderColor, "border-color", "", l10n.T("Border color (hex, e.g., #b4b4b4)"))
	recordCmd.Flags().IntVar(&recordFlags.borderWidth, "border-width", 0, l10n.T("Border width in pixels"))

	// ===== Network Throttling =====
	recordCmd.Flags().Float64Var(&recordFlags.downloadMbps, "download-mbps", 0, l10n.T("Download speed in Mbps (0 = unlimited)"))
	recordCmd.Flags().Float64Var(&recordFlags.uploadMbps, "upload-mbps", 0, l10n.T("Upload speed in Mbps (0 = unlimited)"))

	// ===== CPU Throttling =====
	recordCmd.Flags().Float64Var(&recordFlags.cpuThrottling, "cpu-throttling", 0, l10n.T("CPU slowdown factor (1.0 = no throttling, 4.0 = 4x slower)"))

	// ===== Banner =====
	recordCmd.Flags().StringVar(&recordFlags.credit, "credit", "", l10n.T("Custom text shown in banner (default: loadshow)"))

	// ===== Browser =====
	recordCmd.Flags().BoolVar(&recordFlags.noHeadless, "no-headless", false, l10n.T("Run browser in non-headless mode"))
	recordCmd.Flags().StringVar(&recordFlags.chromePath, "chrome-path", "", l10n.T("Path to Chrome executable"))
	recordCmd.Flags().BoolVar(&recordFlags.ignoreHTTPS, "ignore-https-errors", false, l10n.T("Ignore HTTPS certificate errors"))
	recordCmd.Flags().StringVar(&recordFlags.proxyServer, "proxy-server", "", l10n.T("HTTP proxy server (e.g., http://proxy:8080)"))
	recordCmd.Flags().BoolVar(&recordFlags.noIncognito, "no-incognito", false, l10n.T("Disable incognito mode"))

	// ===== Debug =====
	recordCmd.Flags().BoolVarP(&recordFlags.debug, "debug", "d", false, l10n.T("Enable debug output"))
	recordCmd.Flags().StringVar(&recordFlags.debugDir, "debug-dir", "./debug", l10n.T("Directory for debug output"))

	// ===== Logging =====
	recordCmd.Flags().StringVarP(&recordFlags.logLevel, "log-level", "l", "info", l10n.T("Log level (debug, info, warn, error)"))
	recordCmd.Flags().BoolVar(&recordFlags.quiet, "quiet", false, l10n.T("Suppress all log output"))

	// ===== Juxtapose command flags =====
	juxtaposeCmd.Flags().StringVarP(&juxtaposeFlags.output, "output", "o", "", l10n.T("Output MP4 file path (required)"))
	juxtaposeCmd.Flags().StringVarP(&juxtaposeFlags.quality, "quality", "q", "medium", l10n.T("Quality preset (low, medium, high)"))
	juxtaposeCmd.MarkFlagRequired("output")
}

func runRecord(cmd *cobra.Command, args []string) error {
	url := args[0]

	// Build config from preset and overrides
	cfg := buildRecordConfig()

	// Create logger
	var log ports.Logger
	if recordFlags.quiet {
		log = logger.NewNoop()
	} else {
		log = logger.NewConsole(ports.ParseLogLevel(recordFlags.logLevel))
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
	if recordFlags.debug {
		if err := fs.MkdirAll(recordFlags.debugDir); err != nil {
			return fmt.Errorf("create debug directory: %w", err)
		}
		sink = filesink.New(recordFlags.debugDir, fs, renderer)
	} else {
		sink = nullsink.New()
	}

	// Determine number of workers
	workers := runtime.NumCPU()

	// Create stages
	layoutStage := layout.NewStage()
	recordStage := record.New(browser, sink, log, ports.BrowserOptions{
		Headless:          !recordFlags.noHeadless,
		ChromePath:        recordFlags.chromePath,
		Incognito:         !recordFlags.noIncognito,
		IgnoreHTTPSErrors: recordFlags.ignoreHTTPS,
		ProxyServer:       recordFlags.proxyServer,
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
	orchConfig := cfg.ToOrchestratorConfig(url, recordFlags.output)

	// Print start message
	log.Info(l10n.F("Recording %s (%s preset)...", url, recordFlags.preset))

	// Run pipeline
	if err := orch.Run(ctx, orchConfig); err != nil {
		return err
	}

	log.Info(l10n.F("Output saved to %s", recordFlags.output))
	return nil
}

// buildRecordConfig creates a Config from preset and CLI overrides.
func buildRecordConfig() loadshow.Config {
	// Start with device preset
	var builder *loadshow.ConfigBuilder
	switch recordFlags.preset {
	case "desktop":
		builder = loadshow.NewConfigBuilder()
	default: // mobile is default
		builder = loadshow.NewMobileConfigBuilder()
	}

	// Apply quality preset
	builder.WithQualityPreset(loadshow.QualityPreset(recordFlags.quality))

	// Apply video dimensions
	if recordFlags.width > 0 {
		builder.WithWidth(recordFlags.width)
	}
	if recordFlags.height > 0 {
		builder.WithHeight(recordFlags.height)
	}

	// Apply video output overrides
	if recordFlags.videoCRF > 0 {
		builder.WithVideoCRF(recordFlags.videoCRF)
	}
	if recordFlags.outroMs > 0 {
		builder.WithOutroMs(recordFlags.outroMs)
	}

	// Apply recording overrides
	if recordFlags.screencastQuality > 0 {
		builder.WithScreencastQuality(recordFlags.screencastQuality)
	}
	if recordFlags.viewportWidth > 0 {
		builder.WithViewportWidth(recordFlags.viewportWidth)
	}

	// Apply layout overrides
	if recordFlags.columns > 0 {
		builder.WithColumns(recordFlags.columns)
	}
	if recordFlags.margin > 0 {
		builder.WithMargin(recordFlags.margin)
	}
	if recordFlags.gap > 0 {
		builder.WithGap(recordFlags.gap)
	}
	if recordFlags.indent > 0 {
		builder.WithIndent(recordFlags.indent)
	}
	if recordFlags.outdent > 0 {
		builder.WithOutdent(recordFlags.outdent)
	}

	// Apply style overrides
	if recordFlags.backgroundColor != "" {
		builder.WithBackgroundColor(config.ParseColor(recordFlags.backgroundColor))
	}
	if recordFlags.borderColor != "" {
		builder.WithBorderColor(config.ParseColor(recordFlags.borderColor))
	}
	if recordFlags.borderWidth > 0 {
		builder.WithBorderWidth(recordFlags.borderWidth)
	}

	// Apply network throttling (convert Mbps to bytes/sec)
	if recordFlags.downloadMbps > 0 {
		builder.WithDownloadSpeed(loadshow.MbpsToBytes(recordFlags.downloadMbps))
	}
	if recordFlags.uploadMbps > 0 {
		builder.WithUploadSpeed(loadshow.MbpsToBytes(recordFlags.uploadMbps))
	}

	// Apply CPU throttling
	if recordFlags.cpuThrottling > 0 {
		builder.WithCPUThrottling(recordFlags.cpuThrottling)
	}

	// Apply banner options
	if recordFlags.credit != "" {
		builder.WithCredit(recordFlags.credit)
	}

	// Apply browser options
	if recordFlags.ignoreHTTPS {
		builder.WithIgnoreHTTPSErrors(true)
	}
	if recordFlags.proxyServer != "" {
		builder.WithProxyServer(recordFlags.proxyServer)
	}

	return builder.Build()
}

func runJuxtapose(cmd *cobra.Command, args []string) error {
	left := args[0]
	right := args[1]

	fmt.Println(l10n.T("Juxtapose command not yet implemented."))
	fmt.Println(l10n.F("Would create comparison from %s and %s to %s", left, right, juxtaposeFlags.output))
	return nil
}
