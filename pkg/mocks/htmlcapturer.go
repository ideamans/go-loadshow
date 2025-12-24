package mocks

import (
	"context"
	"image"
)

// HTMLCapturer is a mock implementation of ports.HTMLCapturer.
type HTMLCapturer struct {
	CaptureHTMLFunc             func(ctx context.Context, html string) (image.Image, error)
	CaptureElementFunc          func(ctx context.Context, html string) (image.Image, error)
	CaptureHTMLWithViewportFunc func(ctx context.Context, html string, width, height int) (image.Image, error)

	// Track calls for assertions
	CaptureHTMLCalls             []string
	CaptureElementCalls          []string
	CaptureHTMLWithViewportCalls []struct {
		HTML   string
		Width  int
		Height int
	}
}

// NewHTMLCapturer creates a new mock HTMLCapturer with default behavior.
func NewHTMLCapturer() *HTMLCapturer {
	return &HTMLCapturer{
		CaptureHTMLFunc: func(ctx context.Context, html string) (image.Image, error) {
			return image.NewRGBA(image.Rect(0, 0, 400, 80)), nil
		},
		CaptureElementFunc: func(ctx context.Context, html string) (image.Image, error) {
			return image.NewRGBA(image.Rect(0, 0, 400, 80)), nil
		},
		CaptureHTMLWithViewportFunc: func(ctx context.Context, html string, width, height int) (image.Image, error) {
			return image.NewRGBA(image.Rect(0, 0, width, height)), nil
		},
	}
}

// CaptureHTML implements ports.HTMLCapturer.
func (m *HTMLCapturer) CaptureHTML(ctx context.Context, html string) (image.Image, error) {
	m.CaptureHTMLCalls = append(m.CaptureHTMLCalls, html)
	if m.CaptureHTMLFunc != nil {
		return m.CaptureHTMLFunc(ctx, html)
	}
	return image.NewRGBA(image.Rect(0, 0, 400, 80)), nil
}

// CaptureElement implements ports.HTMLCapturer.
func (m *HTMLCapturer) CaptureElement(ctx context.Context, html string) (image.Image, error) {
	m.CaptureElementCalls = append(m.CaptureElementCalls, html)
	if m.CaptureElementFunc != nil {
		return m.CaptureElementFunc(ctx, html)
	}
	return image.NewRGBA(image.Rect(0, 0, 400, 80)), nil
}

// CaptureHTMLWithViewport implements ports.HTMLCapturer.
func (m *HTMLCapturer) CaptureHTMLWithViewport(ctx context.Context, html string, width, height int) (image.Image, error) {
	m.CaptureHTMLWithViewportCalls = append(m.CaptureHTMLWithViewportCalls, struct {
		HTML   string
		Width  int
		Height int
	}{html, width, height})
	if m.CaptureHTMLWithViewportFunc != nil {
		return m.CaptureHTMLWithViewportFunc(ctx, html, width, height)
	}
	return image.NewRGBA(image.Rect(0, 0, width, height)), nil
}
