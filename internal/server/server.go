// Package server implements a http router that handles creating, listing
// and destroying cloud resources.
package server

// TODO: logging

import (
	"github.com/gorilla/mux"
)

// NewServer creates a new server handling leo-like operations for multiple
// users.
func NewServer() *server {
	r := mux.NewRouter()
	srv := &server{
		router: r,
	}
	return srv
}

// Router returns the gorilla mux router associated with the server.
func (s *server) Router() *mux.Router {
	return s.router
}

type server struct {
	router *mux.Router
}
