// Package main provides the CLI entry point for loadshow.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/ideamans/go-l10n"
	"github.com/urfave/cli/v2"

	"github.com/user/loadshow/pkg/adapters/av1decoder"
	"github.com/user/loadshow/pkg/adapters/av1encoder"
	"github.com/user/loadshow/pkg/adapters/capturehtml"
	"github.com/user/loadshow/pkg/adapters/chromebrowser"
	"github.com/user/loadshow/pkg/adapters/filesink"
	"github.com/user/loadshow/pkg/adapters/ggrenderer"
	"github.com/user/loadshow/pkg/adapters/h264decoder"
	"github.com/user/loadshow/pkg/adapters/h264encoder"
	"github.com/user/loadshow/pkg/adapters/logger"
	"github.com/user/loadshow/pkg/adapters/nullsink"
	"github.com/user/loadshow/pkg/adapters/osfilesystem"
	"github.com/user/loadshow/pkg/config"
	"github.com/user/loadshow/pkg/juxtapose"
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

// Flag category names (will be translated)
// Order is controlled by customCommandHelpTemplate
const (
	catOutput      = "Output"
	catPreset      = "Preset"
	catBrowser     = "Browser"
	catPerformance = "Performance Emulation"
	catLayoutStyle = "Layout and Style"
	catBanner      = "Banner"
	catVideoQuality = "Video and Quality"
	catDebug       = "Debug"
	catLogging     = "Logging"
)

// categoryOrder defines the display order of flag categories
var categoryOrder = []string{
	"Output",
	"Preset",
	"Browser",
	"Performance Emulation",
	"Layout and Style",
	"Banner",
	"Video and Quality",
	"Debug",
	"Logging",
}

// orderedCategories returns categories in the specified order
func orderedCategories(categories []cli.VisibleFlagCategory) []cli.VisibleFlagCategory {
	// Create a map for quick lookup
	catMap := make(map[string]cli.VisibleFlagCategory)
	for _, cat := range categories {
		catMap[cat.Name()] = cat
	}

	// Build ordered list based on translated category names
	var ordered []cli.VisibleFlagCategory
	for _, name := range categoryOrder {
		translatedName := l10n.T(name)
		if cat, ok := catMap[translatedName]; ok {
			ordered = append(ordered, cat)
			delete(catMap, translatedName)
		} else if cat, ok := catMap[name]; ok {
			ordered = append(ordered, cat)
			delete(catMap, name)
		}
	}
	// Append any remaining categories not in our order
	for _, cat := range catMap {
		ordered = append(ordered, cat)
	}
	return ordered
}

func init() {
	// Override help printer for commands
	originalHelpPrinter := cli.HelpPrinter
	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		// Check if this is a command with categories
		if cmd, ok := data.(*cli.Command); ok && len(cmd.VisibleFlagCategories()) > 0 {
			printOrderedCommandHelp(w, cmd)
			return
		}
		originalHelpPrinter(w, templ, data)
	}
}

// printOrderedCommandHelp prints command help with categories in specified order
func printOrderedCommandHelp(w io.Writer, cmd *cli.Command) {
	categories := cmd.VisibleFlagCategories()
	ordered := orderedCategories(categories)

	fmt.Fprintf(w, "NAME:\n   %s - %s\n\n", cmd.HelpName, cmd.Usage)
	fmt.Fprintf(w, "USAGE:\n   %s [command options] %s\n\n", cmd.HelpName, cmd.ArgsUsage)
	fmt.Fprintln(w, "OPTIONS:")

	for _, cat := range ordered {
		fmt.Fprintf(w, "   %s\n\n", cat.Name())
		for _, flag := range cat.Flags() {
			flagStr := strings.TrimSpace(flag.String())
			// Indent multi-line flag descriptions
			lines := strings.Split(flagStr, "\n")
			for i, line := range lines {
				if i == 0 {
					fmt.Fprintf(w, "   %s\n", line)
				} else {
					fmt.Fprintf(w, "      %s\n", strings.TrimSpace(line))
				}
			}
		}
		fmt.Fprintln(w)
	}
}

func main() {
	app := &cli.App{
		Name:    "loadshow",
		Usage:   l10n.T("Create page load videos for web performance visualization"),
		Version: version,
		Commands: []*cli.Command{
			recordCommand(),
			juxtaposeCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func recordCommand() *cli.Command {
	return &cli.Command{
		Name:      "record",
		Usage:     l10n.T("Record a web page loading as MP4 video"),
		ArgsUsage: "<url>",
		Flags: []cli.Flag{
			// ===== 1. Output =====
			&cli.StringFlag{
				Name:     "output",
				Aliases:  []string{"o"},
				Usage:    l10n.T("Output MP4 file path (required)"),
				Required: true,
				Category: l10n.T(catOutput),
			},

			// ===== 2. Preset =====
			&cli.StringFlag{
				Name:     "preset",
				Aliases:  []string{"p"},
				Value:    "mobile",
				Usage:    l10n.T("Device preset (desktop, mobile)"),
				Category: l10n.T(catPreset),
			},
			&cli.StringFlag{
				Name:     "quality",
				Aliases:  []string{"q"},
				Value:    "medium",
				Usage:    l10n.T("Quality preset (low, medium, high)"),
				Category: l10n.T(catPreset),
			},

			// ===== 3. Browser =====
			&cli.IntFlag{
				Name:     "viewport-width",
				Usage:    l10n.T("Browser viewport width (min: 500)"),
				Category: l10n.T(catBrowser),
			},
			&cli.StringFlag{
				Name:     "chrome-path",
				Usage:    l10n.T("Path to Chrome executable"),
				Category: l10n.T(catBrowser),
			},
			&cli.BoolFlag{
				Name:     "no-headless",
				Usage:    l10n.T("Run browser in non-headless mode"),
				Category: l10n.T(catBrowser),
			},
			&cli.BoolFlag{
				Name:     "no-incognito",
				Usage:    l10n.T("Disable incognito mode"),
				Category: l10n.T(catBrowser),
			},
			&cli.BoolFlag{
				Name:     "ignore-https-errors",
				Usage:    l10n.T("Ignore HTTPS certificate errors"),
				Category: l10n.T(catBrowser),
			},
			&cli.StringFlag{
				Name:     "proxy-server",
				Usage:    l10n.T("HTTP proxy server (e.g., http://proxy:8080)"),
				Category: l10n.T(catBrowser),
			},

			// ===== 4. Performance Emulation =====
			&cli.Float64Flag{
				Name:     "download-mbps",
				Usage:    l10n.T("Download speed in Mbps (0 = unlimited)"),
				Category: l10n.T(catPerformance),
			},
			&cli.Float64Flag{
				Name:     "upload-mbps",
				Usage:    l10n.T("Upload speed in Mbps (0 = unlimited)"),
				Category: l10n.T(catPerformance),
			},
			&cli.Float64Flag{
				Name:     "cpu-throttling",
				Usage:    l10n.T("CPU slowdown factor (1.0 = no throttling, 4.0 = 4x slower)"),
				Category: l10n.T(catPerformance),
			},

			// ===== 5. Layout and Style =====
			&cli.IntFlag{
				Name:     "columns",
				Aliases:  []string{"c"},
				Usage:    l10n.T("Number of columns (min: 1)"),
				Category: l10n.T(catLayoutStyle),
			},
			&cli.IntFlag{
				Name:     "margin",
				Usage:    l10n.T("Margin around the canvas in pixels"),
				Category: l10n.T(catLayoutStyle),
			},
			&cli.IntFlag{
				Name:     "gap",
				Usage:    l10n.T("Gap between columns in pixels"),
				Category: l10n.T(catLayoutStyle),
			},
			&cli.IntFlag{
				Name:     "indent",
				Usage:    l10n.T("Additional top margin for columns 2+"),
				Category: l10n.T(catLayoutStyle),
			},
			&cli.IntFlag{
				Name:     "outdent",
				Usage:    l10n.T("Additional bottom margin for column 1"),
				Category: l10n.T(catLayoutStyle),
			},
			&cli.StringFlag{
				Name:     "background-color",
				Usage:    l10n.T("Background color (hex, e.g., #dcdcdc)"),
				Category: l10n.T(catLayoutStyle),
			},
			&cli.StringFlag{
				Name:     "border-color",
				Usage:    l10n.T("Border color (hex, e.g., #b4b4b4)"),
				Category: l10n.T(catLayoutStyle),
			},
			&cli.IntFlag{
				Name:     "border-width",
				Usage:    l10n.T("Border width in pixels"),
				Category: l10n.T(catLayoutStyle),
			},

			// ===== 6. Banner =====
			&cli.StringFlag{
				Name:     "credit",
				Usage:    l10n.T("Custom text shown in banner (default: loadshow)"),
				Category: l10n.T(catBanner),
			},

			// ===== 7. Video and Quality =====
			&cli.BoolFlag{
				Name:     "av1",
				Usage:    l10n.T("Force AV1 codec instead of H.264"),
				Category: l10n.T(catVideoQuality),
			},
			&cli.StringFlag{
				Name:     "ffmpeg-path",
				Usage:    l10n.T("Path to ffmpeg executable (Linux only, for H.264)"),
				Category: l10n.T(catVideoQuality),
			},
			&cli.IntFlag{
				Name:     "width",
				Aliases:  []string{"W"},
				Usage:    l10n.T("Output video width (default: 512)"),
				Category: l10n.T(catVideoQuality),
			},
			&cli.IntFlag{
				Name:     "height",
				Aliases:  []string{"H"},
				Usage:    l10n.T("Output video height (default: 640)"),
				Category: l10n.T(catVideoQuality),
			},
			&cli.IntFlag{
				Name:     "video-crf",
				Usage:    l10n.T("Video CRF value (0-63, lower is better, overrides quality preset)"),
				Category: l10n.T(catVideoQuality),
			},
			&cli.IntFlag{
				Name:     "screencast-quality",
				Usage:    l10n.T("Screencast JPEG quality (0-100, overrides quality preset)"),
				Category: l10n.T(catVideoQuality),
			},
			&cli.IntFlag{
				Name:     "outro-ms",
				Usage:    l10n.T("Duration to hold final frame in milliseconds"),
				Category: l10n.T(catVideoQuality),
			},

			// ===== 8. Debug =====
			&cli.BoolFlag{
				Name:     "debug",
				Aliases:  []string{"d"},
				Usage:    l10n.T("Enable debug output"),
				Category: l10n.T(catDebug),
			},
			&cli.StringFlag{
				Name:     "debug-dir",
				Value:    "./debug",
				Usage:    l10n.T("Directory for debug output"),
				Category: l10n.T(catDebug),
			},

			// ===== 9. Logging =====
			&cli.StringFlag{
				Name:     "log-level",
				Aliases:  []string{"l"},
				Value:    "info",
				Usage:    l10n.T("Log level (debug, info, warn, error)"),
				Category: l10n.T(catLogging),
			},
			&cli.BoolFlag{
				Name:     "quiet",
				Usage:    l10n.T("Suppress all log output"),
				Category: l10n.T(catLogging),
			},
		},
		Action: runRecord,
	}
}

func juxtaposeCommand() *cli.Command {
	return &cli.Command{
		Name:      "juxtapose",
		Usage:     l10n.T("Create a side-by-side comparison video"),
		ArgsUsage: "<left> <right>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "output",
				Aliases:  []string{"o"},
				Usage:    l10n.T("Output MP4 file path (required)"),
				Required: true,
				Category: l10n.T(catOutput),
			},
			&cli.StringFlag{
				Name:     "quality",
				Aliases:  []string{"q"},
				Value:    "medium",
				Usage:    l10n.T("Quality preset (low, medium, high)"),
				Category: l10n.T(catPreset),
			},
			&cli.BoolFlag{
				Name:     "av1",
				Usage:    l10n.T("Force AV1 codec instead of H.264"),
				Category: l10n.T(catVideoQuality),
			},
			&cli.StringFlag{
				Name:     "ffmpeg-path",
				Usage:    l10n.T("Path to ffmpeg executable (Linux only, for H.264)"),
				Category: l10n.T(catVideoQuality),
			},
			&cli.IntFlag{
				Name:     "video-crf",
				Usage:    l10n.T("Video CRF value (0-63, lower is better, overrides quality preset)"),
				Category: l10n.T(catVideoQuality),
			},
			&cli.IntFlag{
				Name:     "gap",
				Value:    10,
				Usage:    l10n.T("Gap between videos in pixels"),
				Category: l10n.T(catLayoutStyle),
			},
		},
		Action: runJuxtapose,
	}
}

func runRecord(c *cli.Context) error {
	if c.NArg() < 1 {
		return errors.New(l10n.T("URL argument is required"))
	}
	url := c.Args().Get(0)

	// Build config from preset and overrides
	cfg := buildRecordConfig(c)

	// Create logger
	var log ports.Logger
	if c.Bool("quiet") {
		log = logger.NewNoop()
	} else {
		log = logger.NewConsole(ports.ParseLogLevel(c.String("log-level")))
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

	// Set custom ffmpeg path if specified (for Linux H.264)
	if ffmpegPath := c.String("ffmpeg-path"); ffmpegPath != "" {
		h264encoder.SetFFmpegPath(ffmpegPath)
		h264decoder.SetFFmpegPath(ffmpegPath)
	}

	// Select encoder: H.264 by default, AV1 if --av1 flag or H.264 not available
	var encoder ports.VideoEncoder
	var codecName string

	if c.Bool("av1") {
		encoder = av1encoder.New()
		codecName = "AV1"
	} else if h264encoder.IsAvailable() {
		encoder = h264encoder.New()
		codecName = "H.264"
	} else {
		// Fallback to AV1 if H.264 is not available (e.g., no ffmpeg on Linux)
		encoder = av1encoder.New()
		codecName = "AV1 (fallback)"
		if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
			log.Warn(l10n.T("ffmpeg not found, falling back to AV1 encoder"))
		}
	}

	// Create debug sink
	var sink ports.DebugSink
	if c.Bool("debug") {
		debugDir := c.String("debug-dir")
		if err := fs.MkdirAll(debugDir); err != nil {
			return fmt.Errorf("create debug directory: %w", err)
		}
		sink = filesink.New(debugDir, fs, renderer)
	} else {
		sink = nullsink.New()
	}

	// Determine number of workers
	workers := runtime.NumCPU()

	// Create stages
	layoutStage := layout.NewStage()
	recordStage := record.New(browser, sink, log, ports.BrowserOptions{
		Headless:          !c.Bool("no-headless"),
		ChromePath:        c.String("chrome-path"),
		Incognito:         !c.Bool("no-incognito"),
		IgnoreHTTPSErrors: c.Bool("ignore-https-errors"),
		ProxyServer:       c.String("proxy-server"),
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
	orchConfig := cfg.ToOrchestratorConfig(url, c.String("output"))

	// Print start message
	log.Info(l10n.F("Recording %s (%s preset, %s codec)...", url, c.String("preset"), codecName))

	// Run pipeline
	if err := orch.Run(ctx, orchConfig); err != nil {
		return err
	}

	log.Info(l10n.F("Output saved to %s", c.String("output")))
	return nil
}

// buildRecordConfig creates a Config from preset and CLI overrides.
func buildRecordConfig(c *cli.Context) loadshow.Config {
	// Start with device preset
	var builder *loadshow.ConfigBuilder
	switch c.String("preset") {
	case "desktop":
		builder = loadshow.NewConfigBuilder()
	default: // mobile is default
		builder = loadshow.NewMobileConfigBuilder()
	}

	// Apply quality preset
	builder.WithQualityPreset(loadshow.QualityPreset(c.String("quality")))

	// Apply video dimensions
	if c.Int("width") > 0 {
		builder.WithWidth(c.Int("width"))
	}
	if c.Int("height") > 0 {
		builder.WithHeight(c.Int("height"))
	}

	// Apply video output overrides
	if c.Int("video-crf") > 0 {
		builder.WithVideoCRF(c.Int("video-crf"))
	}
	if c.Int("outro-ms") > 0 {
		builder.WithOutroMs(c.Int("outro-ms"))
	}

	// Apply recording overrides
	if c.Int("screencast-quality") > 0 {
		builder.WithScreencastQuality(c.Int("screencast-quality"))
	}
	if c.Int("viewport-width") > 0 {
		builder.WithViewportWidth(c.Int("viewport-width"))
	}

	// Apply layout overrides
	if c.Int("columns") > 0 {
		builder.WithColumns(c.Int("columns"))
	}
	if c.Int("margin") > 0 {
		builder.WithMargin(c.Int("margin"))
	}
	if c.Int("gap") > 0 {
		builder.WithGap(c.Int("gap"))
	}
	if c.Int("indent") > 0 {
		builder.WithIndent(c.Int("indent"))
	}
	if c.Int("outdent") > 0 {
		builder.WithOutdent(c.Int("outdent"))
	}

	// Apply style overrides
	if c.String("background-color") != "" {
		builder.WithBackgroundColor(config.ParseColor(c.String("background-color")))
	}
	if c.String("border-color") != "" {
		builder.WithBorderColor(config.ParseColor(c.String("border-color")))
	}
	if c.Int("border-width") > 0 {
		builder.WithBorderWidth(c.Int("border-width"))
	}

	// Apply network throttling (convert Mbps to bytes/sec)
	if c.Float64("download-mbps") > 0 {
		builder.WithDownloadSpeed(loadshow.MbpsToBytes(c.Float64("download-mbps")))
	}
	if c.Float64("upload-mbps") > 0 {
		builder.WithUploadSpeed(loadshow.MbpsToBytes(c.Float64("upload-mbps")))
	}

	// Apply CPU throttling
	if c.Float64("cpu-throttling") > 0 {
		builder.WithCPUThrottling(c.Float64("cpu-throttling"))
	}

	// Apply banner options
	if c.String("credit") != "" {
		builder.WithCredit(c.String("credit"))
	}

	// Apply browser options
	if c.Bool("ignore-https-errors") {
		builder.WithIgnoreHTTPSErrors(true)
	}
	if c.String("proxy-server") != "" {
		builder.WithProxyServer(c.String("proxy-server"))
	}

	return builder.Build()
}

func runJuxtapose(c *cli.Context) error {
	if c.NArg() < 2 {
		return errors.New(l10n.T("Two video arguments are required"))
	}
	left := c.Args().Get(0)
	right := c.Args().Get(1)
	output := c.String("output")

	// Create logger
	log := logger.NewConsole(ports.LevelInfo)

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

	// Determine CRF from quality preset or explicit value
	videoCRF := c.Int("video-crf")
	if videoCRF == 0 {
		// Use quality preset
		qualityPreset := loadshow.QualityPreset(c.String("quality"))
		settings := loadshow.GetQualitySettings(qualityPreset)
		videoCRF = settings.VideoCRF
	}

	// Create adapters
	fs := osfilesystem.New()

	// Set custom ffmpeg path if specified (for Linux H.264)
	if ffmpegPath := c.String("ffmpeg-path"); ffmpegPath != "" {
		h264encoder.SetFFmpegPath(ffmpegPath)
		h264decoder.SetFFmpegPath(ffmpegPath)
	}

	// Select encoder and decoder: H.264 by default, AV1 if --av1 flag or H.264 not available
	var encoder ports.VideoEncoder
	var decoder ports.VideoDecoder
	var codecName string

	if c.Bool("av1") {
		encoder = av1encoder.New()
		decoder = av1decoder.NewMP4Reader()
		codecName = "AV1"
	} else if h264encoder.IsAvailable() {
		encoder = h264encoder.New()
		decoder = h264decoder.NewMP4Reader()
		codecName = "H.264"
	} else {
		// Fallback to AV1 if H.264 is not available (e.g., no ffmpeg on Linux)
		encoder = av1encoder.New()
		decoder = av1decoder.NewMP4Reader()
		codecName = "AV1 (fallback)"
		if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
			log.Warn(l10n.T("ffmpeg not found, falling back to AV1 codec"))
		}
	}

	// Create juxtapose options
	opts := juxtapose.Options{
		Gap:     c.Int("gap"),
		FPS:     30.0,
		Quality: videoCRF,
		Bitrate: 0,
	}

	// Create and run juxtapose stage
	stage := juxtapose.New(decoder, encoder, fs, log, opts)

	log.Info(l10n.F("Creating comparison video: %s + %s → %s", left, right, output))
	log.Info(l10n.F("Encoding video with CRF %d (%s codec)", videoCRF, codecName))

	result, err := stage.Execute(ctx, juxtapose.Input{
		LeftPath:   left,
		RightPath:  right,
		OutputPath: output,
	})
	if err != nil {
		return fmt.Errorf("juxtapose: %w", err)
	}

	log.Info(l10n.F("Output saved to %s", result.OutputPath))
	log.Info(l10n.F("Frames: %d, Duration: %dms", result.FrameCount, result.DurationMs))

	return nil
}
