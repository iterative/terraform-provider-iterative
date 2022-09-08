package resources

import (
	"context"
	"fmt"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
)

// NewExistingBucket creates a new data source referring to a pre-allocated GCP storage bucket.
func NewExistingBucket(clientCredentials string, storageParams common.RemoteStorage) *ExistingBucket {
	return &ExistingBucket{
		clientCredentials: clientCredentials,
		params:            storageParams,
	}
}

// ExistingBucket identifies a pre-allocated storage bucket.
type ExistingBucket struct {
	clientCredentials string
	params            common.RemoteStorage
}

// Read verifies the specified storage bucket exists and is accessible.
func (b *ExistingBucket) Read(ctx context.Context) error {
	connection := b.connection()
	err := machine.CheckStorage(ctx, connection)
	if err != nil {
		return fmt.Errorf("failed to verify storage: %w", err)
	}
	return nil
}

func (b *ExistingBucket) connection() machine.RcloneConnection {
	return machine.RcloneConnection{
		Backend:   machine.RcloneBackendGoogleCloudStorage,
		Container: b.params.Container,
		Path:      b.params.Path,
		Config: map[string]string{
			"service_account_credentials": b.clientCredentials,
		}}
}

// ConnectionString implements common.StorageCredentials.
// The method returns the rclone connection string for the specific bucket.
func (b *ExistingBucket) ConnectionString(ctx context.Context) (string, error) {
	return b.connection().String(), nil
}

var _ common.StorageCredentials = (*ExistingBucket)(nil)
