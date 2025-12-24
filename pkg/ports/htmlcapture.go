package ports

import (
	"context"
	"image"
)

// HTMLCapturer captures HTML as images.
type HTMLCapturer interface {
	// CaptureHTML renders HTML content and returns a screenshot.
	CaptureHTML(ctx context.Context, html string) (image.Image, error)

	// CaptureElement renders HTML and captures only the body element.
	CaptureElement(ctx context.Context, html string) (image.Image, error)

	// CaptureHTMLWithViewport renders HTML at a specific viewport size.
	CaptureHTMLWithViewport(ctx context.Context, html string, width, height int) (image.Image, error)
}
