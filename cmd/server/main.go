package main

import (
	"log"
	"net/http"

	"terraform-provider-iterative/internal/server"
)

const port = ":8080"

func main() {
	srv := server.NewServer()
	h := server.Handler(srv)

	log.Fatal(http.ListenAndServe(port, h))
}
