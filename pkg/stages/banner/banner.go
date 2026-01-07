// Package banner implements the banner generation stage.
package banner

import (
	"context"
	"fmt"

	"github.com/user/loadshow/pkg/pipeline"
	"github.com/user/loadshow/pkg/ports"
)

// Stage generates a banner image displaying page metadata.
type Stage struct {
	capturer ports.HTMLCapturer
	sink     ports.DebugSink
	logger   ports.Logger
}

// NewStage creates a new banner stage.
func NewStage(capturer ports.HTMLCapturer, sink ports.DebugSink, logger ports.Logger) *Stage {
	return &Stage{
		capturer: capturer,
		sink:     sink,
		logger:   logger.WithComponent("banner"),
	}
}

// Execute generates a banner image by rendering HTML template in a browser.
func (s *Stage) Execute(ctx context.Context, input pipeline.BannerInput) (pipeline.BannerResult, error) {
	result := pipeline.BannerResult{}

	s.logger.Debug("Generating banner")

	// Create template variables
	vars := NewTemplateVarsWithTimeout(
		input.Width,
		input.URL,
		input.Title,
		input.LoadTimeMs,
		input.TotalBytes,
		input.Credit,
		input.TimedOut,
		input.TimeoutSec,
	)

	// Render HTML template
	html, err := RenderHTML(vars)
	if err != nil {
		return result, fmt.Errorf("render HTML: %w", err)
	}

	// Capture HTML as image using browser (height is determined by content)
	img, err := s.capturer.CaptureHTMLWithViewport(ctx, html, input.Width, 200)
	if err != nil {
		return result, fmt.Errorf("capture HTML: %w", err)
	}

	result.Image = img
	s.logger.Debug("Banner generated: %dx%d", img.Bounds().Dx(), img.Bounds().Dy())

	// Save to debug sink if enabled
	if s.sink.Enabled() {
		s.sink.SaveBanner(result.Image)
	}

	return result, nil
}
