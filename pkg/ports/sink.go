package ports

import (
	"image"
)

// DebugSink abstracts debug output for intermediate results.
// It allows saving intermediate processing results for debugging purposes.
type DebugSink interface {
	// Enabled returns true if debug output is enabled.
	Enabled() bool

	// SaveLayoutJSON saves the layout calculation result as JSON.
	SaveLayoutJSON(data []byte) error

	// SaveLayoutSVG saves the layout visualization as SVG.
	SaveLayoutSVG(data []byte) error

	// SaveRecordingJSON saves the recording metadata as JSON.
	SaveRecordingJSON(data []byte) error

	// SaveRawFrame saves a raw recording frame.
	SaveRawFrame(index int, data []byte) error

	// SaveBanner saves the generated banner image.
	SaveBanner(img image.Image) error

	// SaveComposedFrame saves a composed frame.
	SaveComposedFrame(index int, img image.Image) error
}
