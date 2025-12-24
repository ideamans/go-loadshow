// Package main is a test program for screencast frame recording.
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

const (
	targetURL     = "https://dummy-ec-site.ideamans.com/"
	framesDir     = "tmp/frames"
	windowWidth   = 500
	windowHeight  = 10000
	recordTimeout = 30 * time.Second
)

type frame struct {
	index   int
	elapsed int64
	data    []byte
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Delete tmp/frames directory
	fmt.Printf("Cleaning up %s...\n", framesDir)
	if err := os.RemoveAll(framesDir); err != nil {
		return fmt.Errorf("failed to remove %s: %w", framesDir, err)
	}
	if err := os.MkdirAll(framesDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %w", framesDir, err)
	}

	// 2. Create browser context with window size only (no viewport override)
	fmt.Printf("Launching browser with window size %dx%d (no viewport override)...\n", windowWidth, windowHeight)

	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-translate", true),
		chromedp.Flag("mute-audio", true),
		// Hide scrollbars
		chromedp.Flag("hide-scrollbars", true),
		// Headless mode
		chromedp.Flag("headless", "new"),
		// Window size only
		chromedp.WindowSize(windowWidth, windowHeight),
		chromedp.Flag("window-size", fmt.Sprintf("%d,%d", windowWidth, windowHeight)),
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set up network and CPU throttling
	fmt.Println("Setting network throttling (10mbps) and CPU throttling (x4)...")
	if err := chromedp.Run(ctx,
		network.Enable(),
		network.EmulateNetworkConditions(
			false,                    // offline
			20,                       // latency (ms)
			10*1024*1024/8,           // download throughput (10mbps in bytes/sec)
			10*1024*1024/8,           // upload throughput (10mbps in bytes/sec)
		),
		emulation.SetCPUThrottlingRate(4), // CPU throttling x4
	); err != nil {
		return fmt.Errorf("failed to set throttling: %w", err)
	}

	// 3. Set up frame capture (store in memory)
	var frames []frame
	frameIndex := 0
	startTime := time.Now()
	done := make(chan struct{})

	chromedp.ListenTarget(ctx, func(ev any) {
		switch e := ev.(type) {
		case *page.EventScreencastFrame:
			data, err := base64.StdEncoding.DecodeString(e.Data)
			if err != nil {
				fmt.Printf("Failed to decode frame %d: %v\n", frameIndex, err)
				return
			}

			elapsed := time.Since(startTime).Milliseconds()
			frames = append(frames, frame{
				index:   frameIndex,
				elapsed: elapsed,
				data:    data,
			})

			fmt.Printf("Captured frame %d at %dms (%d bytes)\n", frameIndex, elapsed, len(data))
			frameIndex++

			// Acknowledge frame
			go chromedp.Run(ctx, page.ScreencastFrameAck(e.SessionID))

		case *page.EventLoadEventFired:
			fmt.Printf("Page load event fired at %dms\n", time.Since(startTime).Milliseconds())
			// Stop screencast after a short delay
			go func() {
				time.Sleep(500 * time.Millisecond)
				chromedp.Run(ctx, page.StopScreencast())
				close(done)
			}()
		}
	})

	// 4. Start screencast with JPEG format (no maxWidth/maxHeight constraints)
	fmt.Println("Starting screencast (JPEG, no size constraints)...")
	if err := chromedp.Run(ctx,
		page.StartScreencast().
			WithFormat(page.ScreencastFormatJpeg).
			WithQuality(80).
			WithEveryNthFrame(1),
	); err != nil {
		return fmt.Errorf("failed to start screencast: %w", err)
	}

	// 5. Navigate to target URL
	fmt.Printf("Navigating to %s...\n", targetURL)
	if err := chromedp.Run(ctx, chromedp.Navigate(targetURL)); err != nil {
		return fmt.Errorf("failed to navigate: %w", err)
	}

	// 6. Wait for completion or timeout
	fmt.Println("Recording frames...")
	select {
	case <-done:
		fmt.Println("Recording complete!")
	case <-time.After(recordTimeout):
		fmt.Println("Recording timed out")
		chromedp.Run(ctx, page.StopScreencast())
	}

	// 7. Write all frames to disk
	fmt.Printf("Writing %d frames to disk...\n", len(frames))
	for _, f := range frames {
		filename := filepath.Join(framesDir, fmt.Sprintf("frame_%04d_%dms.jpg", f.index, f.elapsed))
		if err := os.WriteFile(filename, f.data, 0644); err != nil {
			return fmt.Errorf("failed to write frame %d: %w", f.index, err)
		}
		fmt.Printf("Saved %s\n", filename)
	}

	fmt.Printf("Recorded %d frames to %s\n", len(frames), framesDir)
	return nil
}
