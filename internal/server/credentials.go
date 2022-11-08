package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	"terraform-provider-iterative/task/common"
)

const (
	// CredentialsHeader is the name of the header containing the credentials lookup key.
	CredentialsHeader = "credentials"
)

// RequestWithCloudCredentials wraps the provided handler function
// and extracts credentials from the http request's header. The credentials are passed on to
// further handlers via the context.
func CloudCredentialsFromRequest(req *http.Request) (*common.Credentials, error) {
	credentialsRaw := req.Header.Get(CredentialsHeader)
	if len(credentialsRaw) == 0 {
		return nil, errors.New("empty credentials header")
	}
	credentialsJson := make([]byte, base64.StdEncoding.DecodedLen(len(credentialsRaw)))
	n, err := base64.StdEncoding.Decode(credentialsJson, []byte(credentialsRaw))
	if err != nil {
		return nil, err
	}
	credentialsJson = credentialsJson[:n]
	var credentials common.Credentials
	err = json.Unmarshal([]byte(credentialsJson), &credentials)
	if err != nil {
		return nil, err
	}

	err = credentials.Validate()
	if err != nil {
		return nil, err
	}
	return &credentials, nil
}
