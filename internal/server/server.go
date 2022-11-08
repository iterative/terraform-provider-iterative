// Package server implements a http router that handles creating, listing
// and destroying cloud resources.
package server

// TODO: logging

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewServer creates a new server handling leo-like operations for multiple
// users.
func NewServer() *server {
	r := chi.NewRouter()
	srv := &server{
		router: r,
	}
	return srv
}

// Router returns the chi mux associated with the server.
func (s *server) Router() *chi.Mux {
	return s.router
}

type server struct {
	router *chi.Mux
}

// respondError writes the following error to the response writer.
func respondError(ctx context.Context, w http.ResponseWriter, err error) {
	log.Printf("responding with error: %s", err.Error())
	// TODO: implement error to status code mapping.
	w.WriteHeader(http.StatusInternalServerError)
}
