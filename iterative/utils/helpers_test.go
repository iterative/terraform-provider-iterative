package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIdentifier(t *testing.T) {
	d := generateSchemaData(t, map[string]interface{}{"name": "example"})
	SetId(d)
	require.Regexp(t, "^cml-example-", d.Id())
}
