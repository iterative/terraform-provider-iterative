package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"terraform-provider-iterative/task/common"
)

const (
	// CredentialsHeader is the name of the header containing the credentials lookup key.
	CredentialsHeader = "credentials"
)

// CloudCredentials define the cloud provider credentials and region.
type CloudCredentials struct {
	common.Credentials
	Region   common.Region   `json:"region"`
	Provider common.Provider `json:"provider"`
}

// GetCredentials constructs a common.Credentials struct for use in further cloud operations.
func (c CloudCredentials) GetCredentials() common.Credentials {
	return c.Credentials
}

// Validate checks that the credentials are valid.
func (c CloudCredentials) Validate() error {
	switch c.Provider {
	case common.ProviderAWS:
		if c.AWSCredentials == nil {
			return errors.New("empty credentials")
		}
	case common.ProviderAZ:
		if c.AZCredentials == nil {
			return errors.New("empty credentials")
		}
	case common.ProviderGCP:
		if c.GCPCredentials == nil {
			return errors.New("empty credentials")
		}
	case common.ProviderK8S:
		if c.K8SCredentials == nil {
			return errors.New("empty credentials")
		}
	default:
		return fmt.Errorf("unsupported cloud provider: %q", c.Provider)
	}

	fields := []bool{c.AWSCredentials != nil, c.GCPCredentials != nil, c.AZCredentials != nil, c.K8SCredentials != nil}
	var count int
	for _, fieldNotNil := range fields {
		if fieldNotNil {
			count++
		}
	}
	if count > 1 {
		return errors.New("conflicting credentials")
	}
	return nil
}

// CredentialsFromRequest extracts credentials from the http request's header.
func CredentialsFromRequest(req *http.Request) (*CloudCredentials, error) {
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
	var credentials CloudCredentials
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
