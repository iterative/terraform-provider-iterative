package server

import (
	"errors"
	"net/http"
)

// List lists available task deployments.
func (s *server) List(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	creds, ok := credentialsFromContext(req.Context())
	if !ok {
		return nil, errors.New("no credentials supplied")
	}

	// TODO: run task.List supplying the credentials to it.
	return listResponse{}, nil
}

type listResponse struct {
	Deployments []string `json: "deployments"`
}
