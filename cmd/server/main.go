package main

import (
	"log"
	"net/http"

	"terraform-provider-iterative/internal/server"
)

const port = ":8080"

func main() {
	srv := server.NewServer()
	log.Fatal(http.ListenAndServe(port, srv.Router()))
}
