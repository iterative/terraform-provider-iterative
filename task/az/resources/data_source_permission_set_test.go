package resources_test

import (
	"github.com/iterative/terraform-provider-iterative/task/az/resources"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestARMIDValidation tests the validation of user-assigned identity values.
func TestARMIDValidation(t *testing.T) {
	tests := []struct {
		Identity    string
		ExpectError string
	}{{
		Identity:    "/subscriptions/cse78759-ef49-49f7-b371-f6841fa82182/resourceGroups/resource-group/providers/Microsoft.ManagedIdentity/userAssignedIdentities/managed-identity",
		ExpectError: "",
	}, {
		Identity:    "/subscriptions/no-valid",
		ExpectError: `invalid user-assigned identity id: "/subscriptions/no-valid"`,
	}}

	for _, test := range tests {
		err := resources.ValidateARMID(test.Identity)
		if test.ExpectError == "" {
			require.NoError(t, err)
		} else {
			require.EqualError(t, err, test.ExpectError)
		}
	}
}
