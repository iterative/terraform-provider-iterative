package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	"terraform-provider-iterative/task/common"
)

const (
	// CredentialsHeader is the name of the header containing the credentials lookup key.
	CredentialsHeader = "credentials"
)

// RequestWithCloudCredentials wraps the provided handler function
// and extracts credentials from the http request's header. The credentials are passed on to
// further handlers via the context.
func RequestWithCloudCredentials(h func(w http.ResponseWriter, req *http.Request)) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		credentialsRaw := req.Header.Get(CredentialsHeader)
		if len(credentialsRaw) == 0 {
			respondError(req.Context(), w, errors.New("empty credentials header"))
			return
		}
		credentialsJson := make([]byte, base64.StdEncoding.DecodedLen(len(credentialsRaw)))
		n, err := base64.StdEncoding.Decode(credentialsJson, []byte(credentialsRaw))
		if err != nil {
			respondError(req.Context(), w, err)
			return
		}
		credentialsJson = credentialsJson[:n]
		var credentials common.Credentials
		err = json.Unmarshal([]byte(credentialsJson), &credentials)
		if err != nil {
			respondError(req.Context(), w, err)
			return
		}

		err = credentials.Validate()
		if err != nil {
			respondError(req.Context(), w, err)
			return
		}
		ctx := contextWithCredentials(req.Context(), credentials)
		h(w, req.WithContext(ctx))
	}
}

type contextCredentialsKey struct{}

// contextWithCredentials stores the supplied credentials in the context.
func contextWithCredentials(ctx context.Context, creds common.Credentials) context.Context {
	return context.WithValue(ctx, contextCredentialsKey{}, creds)
}

// CredentialsFromContext retrieves credentials from the context.
func CredentialsFromContext(ctx context.Context) (common.Credentials, bool) {
	creds, ok := ctx.Value(contextCredentialsKey{}).(common.Credentials)
	return creds, ok
}
