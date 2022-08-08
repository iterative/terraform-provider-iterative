package machine_test

import (
	"testing"
	"time"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
)

func TestScript(t *testing.T) {
	g := goldie.New(t, goldie.WithDiffEngine(goldie.ColoredDiff))

	// Test with minimal parameters.
	script := `
#!/bin/sh
echo "done"
`[:1]
	output, err := machine.Script(script, nil, nil, nil)
	g.Assert(t, "machine_script_minimal", []byte(output))

	// Test script generation with full parameters.
	credentials := map[string]string{
		"SECRET": "VALUE",
	}
	value := "VALUE"
	variables := common.Variables{
		"KEY": &value,
	}
	timeout := time.Unix(1659919333, 0)
	output, err = machine.Script(script, credentials, variables, &timeout)
	require.NoError(t, err)
	g.Assert(t, "machine_script_full", []byte(output))
}
