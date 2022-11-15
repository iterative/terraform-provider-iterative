package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

var (
	// ErrNotFound represents a failure to lookup a resource.
	ErrNotFound = errors.New("not found")
)

// ResponseCoder is implemented by error types that specify the http status code the
// server should respond with when encountering that error.
type ResponseCoder interface {
	ResponseCode() int
}

// RespondError writes the following error to the response writer.
func RespondError(ctx context.Context, w http.ResponseWriter, err error) {
	log.Printf("responding with error: %s", err.Error())

	var responseCode int = http.StatusInternalServerError
	// Determine the response code.
	if coder, ok := err.(ResponseCoder); ok {
		responseCode = coder.ResponseCode()
	}
	w.WriteHeader(responseCode)
	response := errorResponse{
		Error: err.Error(),
	}
	werr := json.NewEncoder(w).Encode(response)
	if werr != nil {
		log.Printf("failed to marshal error response: %v", werr)
	}
}

type errorResponse struct {
	Error string `json:"error"`
}
