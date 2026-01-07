// Package chromebrowser provides a browser implementation using chromedp.
package chromebrowser

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"github.com/user/loadshow/pkg/ports"
)

// Browser implements ports.Browser using chromedp.
type Browser struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	ctx         context.Context
	cancel      context.CancelFunc

	screencastChan   chan ports.ScreenFrame
	screencastDone   chan struct{}
	screencastMu     sync.Mutex
	screencastActive bool

	pageInfo   *ports.PageInfo
	pageInfoMu sync.Mutex

	resourceCount   int
	resourceCountMu sync.Mutex
	totalBytes      int64
}

// New creates a new Browser.
func New() *Browser {
	return &Browser{}
}

// Launch starts the browser with the given options.
func (b *Browser) Launch(ctx context.Context, opts ports.BrowserOptions) error {
	// Start with default options but customize headless mode
	chromedpOpts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-translate", true),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
	}

	if opts.Headless {
		// Use new headless mode for better compatibility
		chromedpOpts = append(chromedpOpts, chromedp.Flag("headless", "new"))
	}

	// Hide scrollbars for cleaner screenshots
	chromedpOpts = append(chromedpOpts, chromedp.Flag("hide-scrollbars", true))

	// Resolve Chrome path: CLI option → CHROME_PATH env → system defaults
	chromePath := ResolveChromePath(opts.ChromePath)
	if chromePath == "" {
		return fmt.Errorf("chrome not found: please install Chrome/Chromium, set CHROME_PATH environment variable, or use --chrome-path option")
	}
	chromedpOpts = append(chromedpOpts, chromedp.ExecPath(chromePath))

	// Incognito mode
	if opts.Incognito {
		chromedpOpts = append(chromedpOpts, chromedp.Flag("incognito", true))
	}

	if opts.UserAgent != "" {
		chromedpOpts = append(chromedpOpts, chromedp.UserAgent(opts.UserAgent))
	}

	// Set window size for proper screencast dimensions
	if opts.WindowWidth > 0 && opts.WindowHeight > 0 {
		chromedpOpts = append(chromedpOpts,
			chromedp.WindowSize(opts.WindowWidth, opts.WindowHeight),
			chromedp.Flag("window-size", fmt.Sprintf("%d,%d", opts.WindowWidth, opts.WindowHeight)))
	}

	// Ignore HTTPS certificate errors
	if opts.IgnoreHTTPSErrors {
		chromedpOpts = append(chromedpOpts,
			chromedp.Flag("ignore-certificate-errors", true),
			chromedp.Flag("ignore-certificate-errors-spki-list", true),
			chromedp.Flag("allow-insecure-localhost", true))
	}

	// HTTP proxy server
	if opts.ProxyServer != "" {
		chromedpOpts = append(chromedpOpts,
			chromedp.Flag("proxy-server", opts.ProxyServer))
	}

	// Additional flags for server/background/container execution
	chromedpOpts = append(chromedpOpts,
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("single-process", false),
		chromedp.Flag("disable-setuid-sandbox", true),
		// Additional flags for CI/container environments
		chromedp.Flag("disable-namespace-sandbox", true),
		chromedp.Flag("disable-seccomp-filter-sandbox", true),
		chromedp.Flag("no-zygote", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
	)

	b.allocCtx, b.allocCancel = chromedp.NewExecAllocator(ctx, chromedpOpts...)
	b.ctx, b.cancel = chromedp.NewContext(b.allocCtx)

	// Set custom headers if provided
	if len(opts.Headers) > 0 {
		headers := make(map[string]interface{})
		for k, v := range opts.Headers {
			headers[k] = v
		}
		if err := chromedp.Run(b.ctx, network.SetExtraHTTPHeaders(network.Headers(headers))); err != nil {
			return fmt.Errorf("set headers: %w", err)
		}
	}

	return nil
}

// Navigate loads the specified URL.
func (b *Browser) Navigate(url string) error {
	return chromedp.Run(b.ctx, chromedp.Navigate(url))
}

// SetViewport sets the browser viewport dimensions with device scale factor.
// viewportWidth/viewportHeight are in CSS pixels.
// screenWidth/screenHeight are in device pixels (used for screencast).
// deviceScaleFactor determines how CSS pixels map to device pixels.
func (b *Browser) SetViewport(viewportWidth, viewportHeight, screenWidth, screenHeight int, deviceScaleFactor float64) error {
	// First, set window bounds to control the actual browser window size
	// This is needed before SetDeviceMetricsOverride to ensure proper sizing
	if err := chromedp.Run(b.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		windowID, _, err := browser.GetWindowForTarget().Do(ctx)
		if err != nil {
			return nil // Ignore error, continue with device metrics
		}

		bounds := &browser.Bounds{
			Width:  int64(viewportWidth),
			Height: int64(viewportHeight),
		}
		return browser.SetWindowBounds(windowID, bounds).Do(ctx)
	})); err != nil {
		// Ignore window bounds errors
	}

	// Set device metrics override for viewport emulation
	// - viewport width/height in CSS pixels
	// - screen width/height in device pixels (for screencast output)
	// - deviceScaleFactor maps CSS pixels to device pixels
	// - mobile=true enables viewport meta tag emulation for full page capture
	if err := chromedp.Run(b.ctx,
		emulation.SetDeviceMetricsOverride(int64(viewportWidth), int64(viewportHeight), deviceScaleFactor, true).
			WithScreenWidth(int64(screenWidth)).
			WithScreenHeight(int64(screenHeight)),
	); err != nil {
		return fmt.Errorf("set device metrics: %w", err)
	}

	return nil
}

// SetNetworkConditions configures network throttling.
func (b *Browser) SetNetworkConditions(conditions ports.NetworkConditions) error {
	return chromedp.Run(b.ctx,
		network.Enable(),
		network.EmulateNetworkConditions(
			conditions.Offline,
			float64(conditions.LatencyMs),
			float64(conditions.DownloadSpeed),
			float64(conditions.UploadSpeed),
		),
	)
}

// SetCPUThrottling sets CPU throttling rate.
func (b *Browser) SetCPUThrottling(rate float64) error {
	return chromedp.Run(b.ctx,
		emulation.SetCPUThrottlingRate(rate),
	)
}

// StartScreencast begins capturing screenshots at regular intervals.
// maxWidth/maxHeight constrain the output image dimensions.
func (b *Browser) StartScreencast(quality, maxWidth, maxHeight int) (<-chan ports.ScreenFrame, error) {
	b.screencastMu.Lock()
	defer b.screencastMu.Unlock()

	if b.screencastActive {
		return nil, fmt.Errorf("screencast already active")
	}

	b.screencastChan = make(chan ports.ScreenFrame, 100)
	b.screencastDone = make(chan struct{})
	b.screencastActive = true

	startTime := time.Now()

	// Set up event listener for screencast frames
	chromedp.ListenTarget(b.ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *page.EventScreencastFrame:
			data, err := base64.StdEncoding.DecodeString(e.Data)
			if err != nil {
				return
			}

			frame := ports.ScreenFrame{
				TimestampMs: int(time.Since(startTime).Milliseconds()),
				Data:        data,
				Metadata: ports.ScreenFrameMetadata{
					LoadedResources: b.getResourceCount(),
					TotalBytes:      b.getTotalBytes(),
				},
			}

			// Check if screencast is still active before sending
			b.screencastMu.Lock()
			active := b.screencastActive
			if active {
				select {
				case b.screencastChan <- frame:
				default:
					// Channel full, skip frame
				}
			}
			b.screencastMu.Unlock()

			// Acknowledge frame (do this even if channel is closed)
			go chromedp.Run(b.ctx, page.ScreencastFrameAck(e.SessionID))

		case *network.EventLoadingFinished:
			b.incrementResourceCount()
			b.addBytes(int64(e.EncodedDataLength))

		case *page.EventLoadEventFired:
			// Page fully loaded, stop screencast after a short delay
			go func() {
				time.Sleep(500 * time.Millisecond)
				b.StopScreencast()
			}()
		}
	})

	// Start screencast without size constraints (rely on window size only)
	// Note: maxWidth/maxHeight parameters are ignored to avoid rendering issues
	err := chromedp.Run(b.ctx,
		page.StartScreencast().
			WithFormat(page.ScreencastFormatJpeg).
			WithQuality(int64(quality)).
			WithEveryNthFrame(1),
	)
	if err != nil {
		b.screencastActive = false
		close(b.screencastChan)
		return nil, fmt.Errorf("start screencast: %w", err)
	}

	return b.screencastChan, nil
}

// StopScreencast stops the screencast capture.
func (b *Browser) StopScreencast() error {
	b.screencastMu.Lock()
	defer b.screencastMu.Unlock()

	if !b.screencastActive {
		return nil
	}

	b.screencastActive = false

	// Stop screencast with timeout to prevent hanging
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	chromedp.Run(stopCtx, page.StopScreencast())

	// Close channel
	close(b.screencastChan)

	return nil
}

// GetPageInfo retrieves information about the current page.
func (b *Browser) GetPageInfo() (*ports.PageInfo, error) {
	var title, url string
	var scrollHeight, scrollWidth int

	err := chromedp.Run(b.ctx,
		chromedp.Title(&title),
		chromedp.Location(&url),
		chromedp.Evaluate(`document.body.scrollHeight`, &scrollHeight),
		chromedp.Evaluate(`document.body.scrollWidth`, &scrollWidth),
	)
	if err != nil {
		return nil, fmt.Errorf("get page info: %w", err)
	}

	return &ports.PageInfo{
		Title:        title,
		URL:          url,
		ScrollHeight: scrollHeight,
		ScrollWidth:  scrollWidth,
	}, nil
}

// GetPerformanceTiming retrieves navigation timing metrics using Performance API.
func (b *Browser) GetPerformanceTiming() (*ports.PerformanceTiming, error) {
	var timing struct {
		NavigationStart     int64 `json:"navigationStart"`
		DOMContentLoadedEnd int64 `json:"domContentLoadedEventEnd"`
		LoadEventEnd        int64 `json:"loadEventEnd"`
	}

	// Use Performance Navigation Timing API (newer) with fallback to legacy API
	script := `
		(function() {
			// Try Navigation Timing Level 2 first
			const entries = performance.getEntriesByType('navigation');
			if (entries.length > 0) {
				const nav = entries[0];
				return {
					navigationStart: 0,
					domContentLoadedEventEnd: Math.round(nav.domContentLoadedEventEnd),
					loadEventEnd: Math.round(nav.loadEventEnd)
				};
			}
			// Fallback to legacy timing API
			const t = performance.timing;
			return {
				navigationStart: t.navigationStart,
				domContentLoadedEventEnd: t.domContentLoadedEventEnd,
				loadEventEnd: t.loadEventEnd
			};
		})()
	`

	err := chromedp.Run(b.ctx, chromedp.Evaluate(script, &timing))
	if err != nil {
		return nil, fmt.Errorf("get performance timing: %w", err)
	}

	return &ports.PerformanceTiming{
		NavigationStart:     timing.NavigationStart,
		DOMContentLoadedEnd: timing.DOMContentLoadedEnd,
		LoadEventEnd:        timing.LoadEventEnd,
	}, nil
}

// Close shuts down the browser.
func (b *Browser) Close() error {
	b.StopScreencast()

	// Cancel browser context first
	if b.cancel != nil {
		b.cancel()
	}

	// Give Chrome a moment to shut down gracefully, then force kill
	time.Sleep(100 * time.Millisecond)

	if b.allocCancel != nil {
		b.allocCancel()
	}

	return nil
}

func (b *Browser) getResourceCount() int {
	b.resourceCountMu.Lock()
	defer b.resourceCountMu.Unlock()
	return b.resourceCount
}

func (b *Browser) incrementResourceCount() {
	b.resourceCountMu.Lock()
	defer b.resourceCountMu.Unlock()
	b.resourceCount++
}

func (b *Browser) getTotalBytes() int64 {
	b.resourceCountMu.Lock()
	defer b.resourceCountMu.Unlock()
	return b.totalBytes
}

func (b *Browser) addBytes(n int64) {
	b.resourceCountMu.Lock()
	defer b.resourceCountMu.Unlock()
	b.totalBytes += n
}

// Ensure Browser implements ports.Browser
var _ ports.Browser = (*Browser)(nil)
