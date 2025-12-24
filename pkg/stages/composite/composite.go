// Package composite implements the frame composition stage.
package composite

import (
	"context"
	"fmt"
	"image"
	"runtime"
	"sort"
	"sync"

	"github.com/user/loadshow/pkg/pipeline"
	"github.com/user/loadshow/pkg/ports"
)

// Stage composes raw frames into final output frames.
type Stage struct {
	renderer   ports.Renderer
	sink       ports.DebugSink
	logger     ports.Logger
	numWorkers int
}

// NewStage creates a new composite stage.
func NewStage(renderer ports.Renderer, sink ports.DebugSink, logger ports.Logger, numWorkers int) *Stage {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	return &Stage{
		renderer:   renderer,
		sink:       sink,
		logger:     logger.WithComponent("composite"),
		numWorkers: numWorkers,
	}
}

// Execute composes all frames.
func (s *Stage) Execute(ctx context.Context, input pipeline.CompositeInput) (pipeline.CompositeResult, error) {
	if len(input.RawFrames) == 0 {
		return pipeline.CompositeResult{Frames: []pipeline.ComposedFrame{}}, nil
	}

	s.logger.Debug("Compositing %d frames with %d workers", len(input.RawFrames), s.numWorkers)

	// Use parallel processing
	result, err := s.executeParallel(ctx, input)
	if err != nil {
		return result, err
	}

	s.logger.Debug("Composition completed")
	return result, nil
}

// indexedFrame holds a frame with its original index for sorting.
type indexedFrame struct {
	index int
	frame pipeline.ComposedFrame
}

// executeParallel composes frames using worker pool.
func (s *Stage) executeParallel(ctx context.Context, input pipeline.CompositeInput) (pipeline.CompositeResult, error) {
	numFrames := len(input.RawFrames)
	jobs := make(chan int, numFrames)
	results := make(chan indexedFrame, numFrames)
	errChan := make(chan error, s.numWorkers)

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < s.numWorkers; w++ {
		wg.Add(1)
		go s.worker(ctx, &wg, input, jobs, results, errChan)
	}

	// Send jobs
	for i := 0; i < numFrames; i++ {
		jobs <- i
	}
	close(jobs)

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(results)
		close(errChan)
	}()

	// Collect results
	frames := make([]indexedFrame, 0, numFrames)
	for result := range results {
		frames = append(frames, result)

		// Save debug output if enabled
		if s.sink.Enabled() {
			s.sink.SaveComposedFrame(result.index, result.frame.Image)
		}
	}

	// Check for errors
	if err := <-errChan; err != nil {
		return pipeline.CompositeResult{}, err
	}

	// Sort by index to maintain order
	sort.Slice(frames, func(i, j int) bool {
		return frames[i].index < frames[j].index
	})

	// Extract frames in order
	composedFrames := make([]pipeline.ComposedFrame, len(frames))
	for i, f := range frames {
		composedFrames[i] = f.frame
	}

	return pipeline.CompositeResult{Frames: composedFrames}, nil
}

// worker processes frames from jobs channel.
func (s *Stage) worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	input pipeline.CompositeInput,
	jobs <-chan int,
	results chan<- indexedFrame,
	errChan chan<- error,
) {
	defer wg.Done()

	for idx := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		frame, err := s.composeFrame(input, idx)
		if err != nil {
			select {
			case errChan <- fmt.Errorf("compose frame %d: %w", idx, err):
			default:
			}
			return
		}

		results <- indexedFrame{index: idx, frame: frame}
	}
}

// composeFrame creates a single composed frame.
// Frame structure (from top to bottom): Banner → Progress bar → Content
// Following TypeScript composition.ts exactly.
func (s *Stage) composeFrame(input pipeline.CompositeInput, frameIndex int) (pipeline.ComposedFrame, error) {
	rawFrame := input.RawFrames[frameIndex]
	layout := input.Layout

	// Frame dimensions (TypeScript: bannerHeight + progressHeight + canvasHeight)
	// canvasHeight in TypeScript is the content area height from layout input
	canvasWidth := layout.ContentArea.X*2 + layout.ContentArea.Width // = canvasWidth from input

	// Use actual banner image height if banner exists, otherwise use layout value
	bannerHeight := layout.BannerArea.Height
	if input.Banner != nil && input.Banner.Image != nil {
		bannerHeight = input.Banner.Image.Bounds().Dy()
	}

	progressHeight := layout.ProgressArea.Height
	contentHeight := layout.ContentArea.Height + layout.ContentArea.Y*2 // = canvasHeight from input

	// Total frame height = banner + progress + content
	rawFrameHeight := bannerHeight + progressHeight + contentHeight
	// Round up to nearest even number for video encoding compatibility
	canvasHeight := (rawFrameHeight + 1) / 2 * 2

	// canvasOffset = bannerHeight + progressHeight (offset for content area)
	canvasOffset := bannerHeight + progressHeight

	// Create canvas
	canvas := s.renderer.CreateCanvas(canvasWidth, canvasHeight, input.Theme.BackgroundColor)

	// Decode the raw frame image
	frameImg, err := s.renderer.DecodeImage(rawFrame.ImageData, ports.FormatJPEG)
	if err != nil {
		return pipeline.ComposedFrame{}, fmt.Errorf("decode frame image: %w", err)
	}

	// Resize the frame to scroll width, maintaining aspect ratio
	// TypeScript: Sharp.resize(input.layoutOutput.scroll.width)
	originalBounds := frameImg.Bounds()
	if layout.Scroll.Width > 0 && originalBounds.Dx() > 0 {
		targetWidth := layout.Scroll.Width
		targetHeight := originalBounds.Dy() * targetWidth / originalBounds.Dx()
		frameImg = s.renderer.ResizeImage(frameImg, targetWidth, targetHeight)

		// Crop to layout.Scroll.Height if the resized image is taller than expected
		resizedBounds := frameImg.Bounds()
		if resizedBounds.Dy() > layout.Scroll.Height {
			frameImg = extractSubImage(frameImg, 0, 0, resizedBounds.Dx(), layout.Scroll.Height)
		}
	}

	// Draw banner if present (at top: 0)
	if input.Banner != nil && input.Banner.Image != nil && bannerHeight > 0 {
		canvas.DrawImage(input.Banner.Image, 0, 0)
	}

	// Draw progress bar if enabled (at top: bannerHeight, just below banner)
	// Progress is based on traffic (bytes downloaded)
	if input.ShowProgress && progressHeight > 0 && input.TotalBytes > 0 {
		// Last frame should always show 100% progress
		isLastFrame := frameIndex == len(input.RawFrames)-1
		var progress float64
		if isLastFrame {
			progress = 1.0
		} else {
			progress = float64(rawFrame.TotalBytes) / float64(input.TotalBytes)
			if progress > 1.0 {
				progress = 1.0
			}
		}

		// Background (full width at bannerHeight)
		canvas.DrawRect(
			0,
			bannerHeight,
			canvasWidth,
			progressHeight,
			input.Theme.ProgressBgColor,
		)

		// Progress fill
		fillWidth := int(float64(canvasWidth) * progress)
		if fillWidth > 0 {
			canvas.DrawRect(
				0,
				bannerHeight,
				fillWidth,
				progressHeight,
				input.Theme.ProgressBarColor,
			)
		}

		// Draw percentage text at right edge (white text)
		percentText := fmt.Sprintf("%d%%", int(progress*100))
		textStyle := ports.TextStyle{
			FontSize: float64(progressHeight) * 0.7,
			Color:    image.White,
			Align:    ports.AlignRight,
		}
		// Measure text height and calculate vertical center offset
		_, textHeight := canvas.MeasureText(percentText, textStyle)
		offset := (float64(progressHeight) - textHeight) / 2.0
		textY := bannerHeight + int(offset+textHeight/2)
		canvas.DrawText(percentText, canvasWidth-4, textY, textStyle)
	}

	// Draw column borders (at column.y + canvasOffset)
	for _, col := range layout.Columns {
		canvas.DrawRectStroke(col.X, col.Y+canvasOffset, col.Width, col.Height, input.Theme.BorderColor, 1)
	}

	// Draw each window from the frame
	// Extract from (0, window.scrollTop) with window dimensions
	// Draw at (window.x, canvasOffset + window.y)
	resizedBounds := frameImg.Bounds()
	for _, window := range layout.Windows {
		// Calculate available height from this scroll position
		availableHeight := resizedBounds.Dy() - window.ScrollTop
		if availableHeight <= 0 {
			break
		}
		height := window.Height
		if height > availableHeight {
			height = availableHeight
		}

		subImg := extractSubImage(frameImg, 0, window.ScrollTop, window.Width, height)
		if subImg != nil {
			// Draw at canvasOffset + window.y (not just window.y)
			canvas.DrawImage(subImg, window.X, canvasOffset+window.Y)
		}

		if height < window.Height {
			break
		}
	}

	return pipeline.ComposedFrame{
		TimestampMs: rawFrame.TimestampMs,
		Image:       canvas.ToImage(),
	}, nil
}

// extractSubImage extracts a portion of an image.
// IMPORTANT: Returns an image with bounds starting at (0,0) for compatibility
// with drawing libraries like gg that may not handle non-zero bounds correctly.
func extractSubImage(img image.Image, x, y, width, height int) image.Image {
	bounds := img.Bounds()

	// Adjust x, y to be relative to image bounds
	srcX := bounds.Min.X + x
	srcY := bounds.Min.Y + y

	// Clamp to image bounds
	if srcX < bounds.Min.X {
		srcX = bounds.Min.X
	}
	if srcY < bounds.Min.Y {
		srcY = bounds.Min.Y
	}
	if srcX+width > bounds.Max.X {
		width = bounds.Max.X - srcX
	}
	if srcY+height > bounds.Max.Y {
		height = bounds.Max.Y - srcY
	}

	if width <= 0 || height <= 0 {
		return nil
	}

	// Always create a new image with bounds starting at (0,0)
	// This ensures compatibility with gg.DrawImage which expects (0,0) origin
	result := image.NewRGBA(image.Rect(0, 0, width, height))
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			result.Set(dx, dy, img.At(srcX+dx, srcY+dy))
		}
	}
	return result
}
