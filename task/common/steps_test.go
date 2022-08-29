package common_test

import (
	"context"
	"errors"
	"testing"

	"terraform-provider-iterative/task/common"

	"github.com/stretchr/testify/require"
)

func TestSteps(t *testing.T) {
	ctx := context.Background()
	stepsRun := []int{}
	steps := []common.Step{{
		Description: "step 1",
		Action: func(context.Context) error {
			stepsRun = append(stepsRun, 1)
			return nil
		},
	}, {
		Description: "step 2",
		// Step 2 returns an error.
		Action: func(context.Context) error {
			stepsRun = append(stepsRun, 2)
			return errors.New("some error")
		},
	}, {
		Description: "step 3",
		Action: func(context.Context) error {
			stepsRun = append(stepsRun, 3)
			return nil
		},
	}}

	// Run the only steps 1 and 3.
	err := common.RunSteps(ctx, []common.Step{steps[0], steps[2]})
	require.NoError(t, err)
	require.Equal(t, stepsRun, []int{1, 3})

	// Run the original test set. Since step 2 returns an error, step 3 won't be run.
	stepsRun = []int{}
	err = common.RunSteps(ctx, steps)
	require.EqualError(t, err, "some error")
	require.Equal(t, stepsRun, []int{1, 2})
}
