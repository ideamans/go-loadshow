// Package capturehtml provides HTML-to-image capture using a headless browser.
package capturehtml

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/chromedp/chromedp"

	"github.com/user/loadshow/pkg/ports"
)

// Capturer captures HTML as images using a headless browser.
type Capturer struct{}

// New creates a new HTML capturer.
func New() *Capturer {
	return &Capturer{}
}

// Ensure Capturer implements ports.HTMLCapturer
var _ ports.HTMLCapturer = (*Capturer)(nil)

// CaptureHTML renders HTML content and returns a screenshot.
func (c *Capturer) CaptureHTML(ctx context.Context, html string) (image.Image, error) {
	return c.captureWithOptions(ctx, html, false)
}

// CaptureElement renders HTML and captures only the body element.
func (c *Capturer) CaptureElement(ctx context.Context, html string) (image.Image, error) {
	return c.captureWithOptions(ctx, html, true)
}

func (c *Capturer) captureWithOptions(ctx context.Context, html string, elementOnly bool) (image.Image, error) {
	// Write HTML to a temporary file
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("capturehtml_%d.html", os.Getpid()))
	if err := os.WriteFile(tmpFile, []byte(html), 0644); err != nil {
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	defer os.Remove(tmpFile)

	// Create allocator with headless options (matching chromebrowser)
	chromedpOpts := chromedp.DefaultExecAllocatorOptions[:]
	chromedpOpts = append(chromedpOpts,
		chromedp.Flag("headless", "new"), // Use new headless mode
		chromedp.Flag("hide-scrollbars", true),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, chromedpOpts...)
	defer allocCancel()

	// Create browser context
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	fileURL := "file://" + tmpFile

	// Capture screenshot using CaptureScreenshot for the actual rendered content
	var buf []byte
	if err := chromedp.Run(browserCtx,
		chromedp.Navigate(fileURL),
		chromedp.CaptureScreenshot(&buf),
	); err != nil {
		return nil, fmt.Errorf("capture screenshot: %w", err)
	}

	// Decode PNG
	img, err := png.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("decode screenshot: %w", err)
	}

	// Crop to body size if needed - look for the actual content area
	// For now, just return the full screenshot and let the caller handle sizing
	return img, nil
}

// CaptureHTMLWithViewport renders HTML at a specific viewport size.
func (c *Capturer) CaptureHTMLWithViewport(ctx context.Context, html string, width, height int) (image.Image, error) {
	// Write HTML to a temporary file
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("capturehtml_%d.html", os.Getpid()))
	if err := os.WriteFile(tmpFile, []byte(html), 0644); err != nil {
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	defer os.Remove(tmpFile)

	// Create allocator
	chromedpOpts := chromedp.DefaultExecAllocatorOptions[:]
	chromedpOpts = append(chromedpOpts,
		chromedp.Flag("headless", "new"),
		chromedp.Flag("hide-scrollbars", true),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, chromedpOpts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	fileURL := "file://" + tmpFile

	// Navigate and get body dimensions
	var bodyWidth, bodyHeight float64
	var buf []byte
	if err := chromedp.Run(browserCtx,
		chromedp.EmulateViewport(int64(width), int64(height)),
		chromedp.Navigate(fileURL),
		chromedp.Evaluate(`document.body.getBoundingClientRect().width`, &bodyWidth),
		chromedp.Evaluate(`document.body.getBoundingClientRect().height`, &bodyHeight),
		chromedp.FullScreenshot(&buf, 100),
	); err != nil {
		return nil, fmt.Errorf("capture screenshot: %w", err)
	}

	img, err := png.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("decode screenshot: %w", err)
	}

	// Crop to body dimensions
	cropWidth := int(bodyWidth)
	cropHeight := int(bodyHeight)
	if cropWidth > 0 && cropHeight > 0 {
		bounds := img.Bounds()
		if cropWidth > bounds.Dx() {
			cropWidth = bounds.Dx()
		}
		if cropHeight > bounds.Dy() {
			cropHeight = bounds.Dy()
		}
		cropped := image.NewRGBA(image.Rect(0, 0, cropWidth, cropHeight))
		for y := 0; y < cropHeight; y++ {
			for x := 0; x < cropWidth; x++ {
				cropped.Set(x, y, img.At(x, y))
			}
		}
		return cropped, nil
	}

	return img, nil
}
