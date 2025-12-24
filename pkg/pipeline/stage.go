// Package pipeline provides the pipeline infrastructure for loadshow.
package pipeline

import (
	"context"
)

// Stage represents a processing stage in the pipeline.
// Each stage takes an input and produces an output.
type Stage[In, Out any] interface {
	// Execute runs the stage with the given input and returns the output.
	Execute(ctx context.Context, input In) (Out, error)
}

// StageFunc is a function adapter for Stage interface.
type StageFunc[In, Out any] func(ctx context.Context, input In) (Out, error)

// Execute implements Stage interface.
func (f StageFunc[In, Out]) Execute(ctx context.Context, input In) (Out, error) {
	return f(ctx, input)
}
