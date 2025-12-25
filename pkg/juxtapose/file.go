package juxtapose

import (
	"context"

	"github.com/user/loadshow/pkg/adapters/av1decoder"
	"github.com/user/loadshow/pkg/adapters/av1encoder"
	"github.com/user/loadshow/pkg/adapters/logger"
	"github.com/user/loadshow/pkg/adapters/osfilesystem"
)

// Combine combines two videos side by side.
// This is a convenience function that uses default adapters.
// For custom dependencies (e.g., custom logger), use the Stage API instead.
//
// Example using Stage API with custom logger:
//
//	stage := juxtapose.New(
//	    av1decoder.NewMP4Reader(),
//	    av1encoder.New(),
//	    osfilesystem.New(),
//	    myCustomLogger,
//	    juxtapose.DefaultOptions(),
//	)
//	result, err := stage.Execute(ctx, juxtapose.Input{
//	    LeftPath:   "left.mp4",
//	    RightPath:  "right.mp4",
//	    OutputPath: "output.mp4",
//	})
func Combine(leftPath, rightPath, outputPath string, opts Options) error {
	// Create default adapters
	decoder := av1decoder.NewMP4Reader()
	defer decoder.Close()

	encoder := av1encoder.New()
	fs := osfilesystem.New()
	log := logger.NewNoop()

	// Create stage with default adapters
	stage := New(decoder, encoder, fs, log, opts)

	// Execute
	_, err := stage.Execute(context.Background(), Input{
		LeftPath:   leftPath,
		RightPath:  rightPath,
		OutputPath: outputPath,
	})

	return err
}
