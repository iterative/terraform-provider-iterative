package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/crypto/nacl/box"

	"terraform-provider-iterative/task/common"
)

type authorizationError struct {
	message string
}

// ResponseCode implements the ResponseCoder interface.
func (a authorizationError) ResponseCode() int {
	return http.StatusUnauthorized
}

// Error implements the error interface.
func (a authorizationError) Error() string {
	return a.message
}

// AuthorizationError returns a new authorization error.
func AuthorizationError(msg string) authorizationError {
	return authorizationError{message: msg}
}

const (
	// CredentialsHeader is the name of the header containing the credentials lookup key.
	CredentialsHeader = "Authorization"
)

// PublicKey and PrivateKey are used for libsodium-like sealed box asymmetric encryption of cloud credentials,
// so clients can store and transmit them without exposing their contents.
var PublicKey, PrivateKey *[32]byte

func init() {
	var err error
	if PublicKey, PrivateKey, err = box.GenerateKey(rand.Reader); err != nil {
		panic(err)
	}
}

// CloudCredentials define the cloud provider credentials and region.
type CloudCredentials struct {
	common.Credentials
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
			return AuthorizationError("empty credentials")
		}
	case common.ProviderAZ:
		if c.AZCredentials == nil {
			return AuthorizationError("empty credentials")
		}
	case common.ProviderGCP:
		if c.GCPCredentials == nil {
			return AuthorizationError("empty credentials")
		}
	case common.ProviderK8S:
		if c.K8SCredentials == nil {
			return AuthorizationError("empty credentials")
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
		return AuthorizationError("conflicting credentials")
	}
	return nil
}

// CredentialsFromRequest extracts credentials from the http request's header.
func CredentialsFromRequest(req *http.Request) (*CloudCredentials, error) {
	headerRaw := req.Header.Get(CredentialsHeader)
	if len(headerRaw) == 0 {
		return nil, AuthorizationError("empty credentials header")
	}
	prefix := "Bearer "
	if len(headerRaw) < len(prefix) || !strings.EqualFold(headerRaw[:len(prefix)], prefix) {
		return nil, AuthorizationError("invalid bearer token")
	}

	headerRaw = headerRaw[len(prefix):]
	credentialsBox := make([]byte, base64.StdEncoding.DecodedLen(len(headerRaw)))
	n, err := base64.StdEncoding.Decode(credentialsBox, []byte(headerRaw))
	if err != nil {
		return nil, AuthorizationError(err.Error())
	}

	credentialsJson, ok := box.OpenAnonymous(nil, credentialsBox[:n], PublicKey, PrivateKey)
	if !ok {
		return nil, AuthorizationError("failed to decrypt credentials")
	}

	var credentials CloudCredentials
	err = json.Unmarshal([]byte(credentialsJson), &credentials)
	if err != nil {
		return nil, AuthorizationError(err.Error())
	}

	err = credentials.Validate()
	if err != nil {
		return nil, err
	}
	return &credentials, nil
}
