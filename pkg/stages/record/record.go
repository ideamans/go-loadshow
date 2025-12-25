// Package record implements the page recording stage.
package record

import (
	"context"
	"fmt"
	"time"

	"github.com/user/loadshow/pkg/pipeline"
	"github.com/user/loadshow/pkg/ports"
)

// Stage records a web page loading process using a browser.
type Stage struct {
	browser     ports.Browser
	sink        ports.DebugSink
	logger      ports.Logger
	browserOpts ports.BrowserOptions
}

// New creates a new record stage.
func New(browser ports.Browser, sink ports.DebugSink, logger ports.Logger, opts ports.BrowserOptions) *Stage {
	return &Stage{
		browser:     browser,
		sink:        sink,
		logger:      logger.WithComponent("browser"),
		browserOpts: opts,
	}
}

// minWindowWidth is the minimum window width for Chrome headless mode.
const minWindowWidth = 500

// maxWindowHeight is the maximum window height to avoid Chrome rendering issues.
const maxWindowHeight = 16000

// Execute records the page loading process.
func (s *Stage) Execute(ctx context.Context, input pipeline.RecordInput) (pipeline.RecordResult, error) {
	result := pipeline.RecordResult{
		Frames: make([]pipeline.RawFrame, 0),
	}

	// Calculate window size based on layout aspect ratio
	// No viewport override - rely only on window size for proper rendering
	// Use 1.5x margin to accommodate timing-dependent size variations during capture
	aspectRatio := float64(input.Screen.Height) / float64(input.Screen.Width)

	// Window width: at least minWindowWidth (Chrome headless minimum)
	windowWidth := input.ViewportWidth
	if windowWidth < minWindowWidth {
		windowWidth = minWindowWidth
	}

	// Window height: aspect ratio * 1.2x margin for capture timing variations
	// Capped at maxWindowHeight to avoid Chrome rendering issues
	windowHeight := int(float64(windowWidth) * aspectRatio * 1.2)
	if windowHeight > maxWindowHeight {
		windowHeight = maxWindowHeight
	}

	// Merge browser options with input
	opts := s.browserOpts
	if len(input.Headers) > 0 {
		opts.Headers = input.Headers
	}
	// Set window size only (no viewport override)
	opts.WindowWidth = windowWidth
	opts.WindowHeight = windowHeight
	// Merge browser options from input
	opts.IgnoreHTTPSErrors = input.IgnoreHTTPSErrors
	opts.ProxyServer = input.ProxyServer

	// Launch browser
	if opts.Headless {
		s.logger.Debug("Launching browser in headless mode")
	} else {
		s.logger.Debug("Launching browser in visible mode")
	}
	if err := s.browser.Launch(ctx, opts); err != nil {
		return result, fmt.Errorf("launch browser: %w", err)
	}
	defer func() {
		s.browser.Close()
		s.logger.Debug("Browser closed")
	}()

	// Note: SetViewport is intentionally not called
	// This avoids right-margin rendering issues observed with viewport emulation

	// Set network conditions
	s.logger.Debug("Setting network conditions: %d ms latency, %d bps down, %d bps up",
		input.NetworkConditions.LatencyMs,
		input.NetworkConditions.DownloadSpeed,
		input.NetworkConditions.UploadSpeed)
	if err := s.browser.SetNetworkConditions(input.NetworkConditions); err != nil {
		return result, fmt.Errorf("set network conditions: %w", err)
	}

	// Set CPU throttling
	if input.CPUThrottling > 0 {
		s.logger.Debug("Setting CPU throttling: %.1fx slowdown", input.CPUThrottling)
		if err := s.browser.SetCPUThrottling(input.CPUThrottling); err != nil {
			return result, fmt.Errorf("set CPU throttling: %w", err)
		}
	}

	// Start screencast (window size controls capture dimensions, these params are ignored)
	// Default to 80 (medium quality) if not specified
	screencastQuality := input.ScreencastQuality
	if screencastQuality <= 0 {
		screencastQuality = 80
	}
	s.logger.Debug("Starting screencast with JPEG quality %d", screencastQuality)
	frameChan, err := s.browser.StartScreencast(screencastQuality, windowWidth, windowHeight)
	if err != nil {
		return result, fmt.Errorf("start screencast: %w", err)
	}

	// Create timeout context
	timeout := time.Duration(input.TimeoutMs) * time.Millisecond
	recordCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Start navigation
	s.logger.Debug("Navigating to %s", input.URL)
	navStart := time.Now()
	if err := s.browser.Navigate(input.URL); err != nil {
		return result, fmt.Errorf("navigate: %w", err)
	}

	// Collect frames
	frameIndex := 0
	for {
		select {
		case <-recordCtx.Done():
			// Timeout or context cancelled
			goto done
		case frame, ok := <-frameChan:
			if !ok {
				// Channel closed, recording complete
				goto done
			}

			rawFrame := pipeline.RawFrame{
				TimestampMs:     frame.TimestampMs,
				ImageData:       frame.Data,
				LoadedResources: frame.Metadata.LoadedResources,
				TotalResources:  frame.Metadata.TotalResources,
				TotalBytes:      frame.Metadata.TotalBytes,
			}
			result.Frames = append(result.Frames, rawFrame)

			// Save debug output if enabled
			if s.sink.Enabled() {
				s.sink.SaveRawFrame(frameIndex, frame.Data)
			}
			frameIndex++
		}
	}

done:
	// Stop screencast
	s.browser.StopScreencast()
	s.logger.Debug("Captured %d frames", len(result.Frames))

	// Get page info
	pageInfo, err := s.browser.GetPageInfo()
	if err != nil {
		return result, fmt.Errorf("get page info: %w", err)
	}
	result.PageInfo = *pageInfo

	// Calculate timing
	totalDuration := time.Since(navStart)
	result.Timing = pipeline.TimingInfo{
		TotalDurationMs: int(totalDuration.Milliseconds()),
	}

	// Set load complete time based on last frame
	if len(result.Frames) > 0 {
		lastFrame := result.Frames[len(result.Frames)-1]
		result.Timing.LoadCompleteMs = lastFrame.TimestampMs
	}

	return result, nil
}
