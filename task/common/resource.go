package common

import (
	"context"
)

// Resource defines the interface implemented by deployment resources.
type Resource interface {
	Read(ctx context.Context) error
	Create(ctx context.Context) error
	Delete(ctx context.Context) error
}

// StorageCredentials is an interface implemented by data sources and resources
// that provide access to cloud storage buckets.
type StorageCredentials interface {
	// ConnectionString returns the connection string necessary to access
	// an S3 bucket.
	ConnectionString(ctx context.Context) (string, error)
}
