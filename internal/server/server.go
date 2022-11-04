// Package server implements a http router that handles creating, listing
// and destroying cloud resources.
package server

// TODO: logging

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

// NewServer creates a new server handling leo-like operations for multiple
// users.
func NewServer() *server {
	r := mux.NewRouter()
	srv := &server{
		router:      r,
		credentials: make(map[string]Credentials),
	}
	r.HandleFunc("/credentials", WrapJSONHandler(srv.Credentials))
	return srv
}

// Router returns the gorilla mux router associated with the server.
func (s *server) Router() *mux.Router {
	return s.router
}

type server struct {
	router *mux.Router

	m sync.Mutex

	credentials map[string]Credentials
}

// Credentials handles in-memory credential storage.
func (s *server) Credentials(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method == http.MethodPost {
		return s.storeCredentials(w, req)
	}
	return nil, fmt.Errorf("unsupported method %q", req.Method)
}
