package common

import (
	"context"
)

// BlobContainer defines the interface implemented by blob container resources
// across cloud providers.
type BlobContainer interface {
	Resource
	// Identifier returns the identifier of the container.
	Identifier() string
}

// Resource defines the interface implemented by deployment resources.
type Resource interface {
	Read(ctx context.Context) error
	Create(ctx context.Context) error
	Delete(ctx context.Context) error
}
