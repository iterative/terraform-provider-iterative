package server

import (
	"encoding/json"
	"log"
	"net/http"
)

// JSONHandler is any http handler function that returns a json response.
// The returned value will be marshalled to json and written to the response.
type JSONHandler func(http.ResponseWriter, *http.Request) (interface{}, error)

// WrapJSONHandler wraps the provided JSONHandler
// The returned handler function will marshal the original return to JSON and
// write it to the response.
func WrapJSONHandler(f JSONHandler) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		ret, err := f(w, req)
		if err != nil {
			log.Printf("error: %s", err.Error())
			// TODO: implement error marshalling and mapping to http status codes.
			w.WriteHeader(http.StatusBadRequest)
		}
		encoder := json.NewEncoder(w)
		err = encoder.Encode(ret)
		if err != nil {
			log.Printf("failed to marshal response to json: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
