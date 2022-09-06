package resources

import (
	"context"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
)

// NewExistingBlobContainer returns a new data source refering to a pre-allocated storage container.
// The containerRef parameter is expected to be in the form of "storageAccountName/containerName".
func NewExistingBlobContainer(client *client.Client, storageParams common.RemoteStorage) *ExistingBlobContainer {
	return &ExistingBlobContainer{
		client: client,
		params: storageParams,
	}
}

// ExistingBlobContainer is a data source referencing an existing azure storage container.
type ExistingBlobContainer struct {
	client *client.Client

	params common.RemoteStorage
}

// Read verifies the specified container exists and retrieves its access key.
func (b *ExistingBlobContainer) Read(ctx context.Context) error {
	conn := b.connection()
	return machine.CheckStorage(ctx, conn)
}

func (b *ExistingBlobContainer) connection() machine.RcloneConnection {
	return machine.RcloneConnection{
		Backend:   machine.RcloneBackendAzureBlob,
		Container: b.params.Container,
		Path:      b.params.Path,
		Config:    b.params.Config,
	}
}

// ConnectionString implements common.StorageCredentials.
// The method returns the rclone connection string for the specific bucket.
func (b *ExistingBlobContainer) ConnectionString(ctx context.Context) (string, error) {
	connection := b.connection()
	return connection.String(), nil
}
