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
		creds, err := server.CloudCredentialsFromRequest(req)
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

	creds := common.Credentials{
		Provider: common.ProviderAWS,
		AWSCredentials: &common.AWSCredentials{
			AccessKeyID:     "aws-access-key",
			SecretAccessKey: "secret",
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
	var response common.Credentials
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.EqualValues(t, response, creds)
}
