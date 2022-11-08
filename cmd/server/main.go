package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"terraform-provider-iterative/internal/server"
)

const port = ":8080"

func main() {
	srv := server.NewServer()
	r := srv.Router()
	r.Use(middleware.Logger)
	log.Fatal(http.ListenAndServe(port, r))
}
