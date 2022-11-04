package server

import (
	"errors"
)

var (
	// ErrNotFound represents a failure to lookup a resource.
	ErrNotFound = errors.New("not found")
)
