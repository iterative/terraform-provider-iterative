package server_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"terraform-provider-iterative/internal/server"
)

func TestCredentialStorage(t *testing.T) {
	srv := server.NewServer()

	httpsrv := httptest.NewServer(srv.Router())
	defer httpsrv.Close()
	client := httpsrv.Client()

	creds := server.Credentials{
		Type: server.CredentialsTypeAWS,
		AWSCredentials: &server.AWSCredentials{
			AccessKeyId:     "aws-access-key",
			SecretAccessKey: "secret",
		},
	}
	buff := &bytes.Buffer{}
	err := json.NewEncoder(buff).Encode(creds)
	assert.NoError(t, err)

	resp, err := client.Post(httpsrv.URL+"/credentials", "application/json", buff)
	assert.NoError(t, err)
	defer resp.Body.Close()
	var response struct{ Key string }
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, response.Key, creds.AWSCredentials.AccessKeyId)

	storedCreds, err := srv.LookupCredentials(response.Key)
	assert.NoError(t, err)
	assert.EqualValues(t, creds, *storedCreds)
}
