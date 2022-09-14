package machine_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"terraform-provider-iterative/task/common/machine"
)

func TestRcloneConnectionString(t *testing.T) {
	tests := []struct {
		description string
		conn        machine.RcloneConnection
		expected    string
	}{{
		description: "connection string with config",
		conn: machine.RcloneConnection{
			Backend:   machine.RcloneBackendAzureBlob,
			Container: "container",
			Config: map[string]string{
				"account": "az_account",
				"key":     "az_key",
			},
		},
		expected: ":azureblob,account='az_account',key='az_key':container",
	}, {
		description: "connection string with path",
		conn: machine.RcloneConnection{
			Backend:   machine.RcloneBackendAzureBlob,
			Container: "container",
			Path:      "/subdirectory",
		},
		expected: ":azureblob:container/subdirectory",
	}, {
		description: "connection string with path, no separator prefix",
		conn: machine.RcloneConnection{
			Backend:   machine.RcloneBackendAzureBlob,
			Container: "container",
			Path:      "subdirectory",
		},
		expected: ":azureblob:container/subdirectory",
	}}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			require.Equal(t, test.expected, test.conn.String())
		})

	}
}
