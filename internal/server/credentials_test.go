package server_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"terraform-provider-iterative/internal/server"
	"terraform-provider-iterative/task/common"
)

func TestCredentialMiddleware(t *testing.T) {
	echoHandler := func(w http.ResponseWriter, req *http.Request) {
		creds, err := server.CredentialsFromRequest(req)
		if err != nil {
			server.RespondError(req.Context(), w, err)
			return
		}
		err = json.NewEncoder(w).Encode(*creds)
		if err != nil {
			server.RespondError(req.Context(), w, err)
			return
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", echoHandler)

	httpsrv := httptest.NewServer(mux)
	defer httpsrv.Close()
	client := httpsrv.Client()

	creds := server.CloudCredentials{
		Provider: common.ProviderAWS,
		Credentials: common.Credentials{
			AWSCredentials: &common.AWSCredentials{
				AccessKeyID:     "aws-access-key",
				SecretAccessKey: "secret",
			},
		},
	}
	buff := &bytes.Buffer{}
	err := json.NewEncoder(buff).Encode(creds)
	assert.NoError(t, err)
	encodedCredentials := base64.StdEncoding.EncodeToString(buff.Bytes())

	req, err := http.NewRequest("GET", httpsrv.URL+"/", nil)
	assert.NoError(t, err)
	req.Header.Set(server.CredentialsHeader, encodedCredentials)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	var response server.CloudCredentials
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.EqualValues(t, response, creds)
}

func TestCredentialValidation(t *testing.T) {
	tests := []struct {
		description string
		credentials server.CloudCredentials
		expectError string
	}{{
		description: "empty credentials",
		credentials: server.CloudCredentials{},
		expectError: `unsupported cloud provider: ""`,
	}, {
		description: "valid AWS credentials",
		credentials: server.CloudCredentials{
			Region:   common.Region("us-east"),
			Provider: common.ProviderAWS,
			Credentials: common.Credentials{
				AWSCredentials: &common.AWSCredentials{},
			},
		},
	}, {
		description: "empty AWS credentials",
		credentials: server.CloudCredentials{
			Provider: common.ProviderAWS,
		},
		expectError: "empty credentials",
	}, {
		description: "conflicting credentials",
		credentials: server.CloudCredentials{
			Provider: common.ProviderAWS,
			Credentials: common.Credentials{
				AWSCredentials: &common.AWSCredentials{},
				AZCredentials:  &common.AZCredentials{},
			},
		},
		expectError: "conflicting credentials",
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
