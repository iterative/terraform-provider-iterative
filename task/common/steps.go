package common

import (
	"context"

	"github.com/0x2b3bfa0/logrusctx"
)

// Step defines a single resource creation step.
type Step struct {
	Action      func(ctx context.Context) error
	Description string
}

// RunSteps executes the specified resource allocation steps.
func RunSteps(ctx context.Context, steps []Step) error {
	total := len(steps)
	for i, step := range steps {
		logrusctx.Infof(ctx, "[%d/%d] %s", i+1, total, step.Description)
		if err := step.Action(ctx); err != nil {
			return err
		}
	}
	return nil
}
