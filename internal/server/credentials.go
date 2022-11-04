package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const (
	CredentialsTypeAWS = "aws"
)

// storeCredentials stores the supplied credentials and returns
// a response indicating a key to reference those credentials in subsequent requests.
func (s *server) storeCredentials(w http.ResponseWriter, req *http.Request) (*storeCredentialsResponse, error) {
	dec := json.NewDecoder(req.Body)
	defer req.Body.Close()

	var creds Credentials
	err := dec.Decode(&creds)
	if err != nil {
		log.Printf("failed to unmarshal request: %s", err.Error())
		return nil, err
	}
	// TODO: more validation
	var key string
	switch creds.Type {
	case CredentialsTypeAWS:
		key = creds.AWSCredentials.AccessKeyId
	default:
		log.Printf("credentials type %q is not supported", creds.Type)
		return nil, fmt.Errorf("unsupported credential type %q", creds.Type)
	}
	s.m.Lock()
	defer s.m.Unlock()
	s.credentials[key] = creds
	return &storeCredentialsResponse{Key: key}, nil
}

// LookupCredentials searches the in-memory store for credentials associated with the
// specified key.
func (s *server) LookupCredentials(key string) (*Credentials, error) {
	s.m.Lock()
	defer s.m.Unlock()
	creds, ok := s.credentials[key]
	if !ok {
		return nil, ErrNotFound
	}
	return &creds, nil
}

// storeCredentialsResponse is returned on succesful requests to store credentials.
type storeCredentialsResponse struct {
	Key string `json:"key"`
}

// Credentials is used to unmarshal the json request payload.
type Credentials struct {
	Type           string          // aws, gcp or az
	AWSCredentials *AWSCredentials `json:"aws",omitempty`
}

// AWSCredentials stores credentials for provisioning AWS resources.
type AWSCredentials struct {
	AccessKeyId     string `json:"aws_access_key_id"`
	SecretAccessKey string `json:"aws_secret_access_key"`
	SessionToken    string `json:"aws_session_token"`
}
