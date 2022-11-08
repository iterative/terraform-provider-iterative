package common_test

import (
	"testing"

	"terraform-provider-iterative/task/common"

	"github.com/stretchr/testify/assert"
)

func TestCredentialValidation(t *testing.T) {
	tests := []struct {
		description string
		credentials common.Credentials
		expectError string
	}{{
		description: "empty credentials",
		credentials: common.Credentials{},
		expectError: "empty credentials",
	}}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			err := test.credentials.Validate()
			if test.expectError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.expectError)
			}
		})
	}

}
