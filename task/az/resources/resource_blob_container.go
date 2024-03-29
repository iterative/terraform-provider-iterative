package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
)

func NewBlobContainer(client *client.Client, identifier common.Identifier, resourceGroup *ResourceGroup, storageAccount *StorageAccount) *BlobContainer {
	b := &BlobContainer{
		client:     client,
		Identifier: identifier.Short(),
	}

	b.Dependencies.ResourceGroup = resourceGroup
	b.Dependencies.StorageAccount = storageAccount
	return b
}

type BlobContainer struct {
	client       *client.Client
	Identifier   string
	Dependencies struct {
		ResourceGroup  *ResourceGroup
		StorageAccount *StorageAccount
	}
	Resource *storage.BlobContainer
}

func (b *BlobContainer) Create(ctx context.Context) error {
	container, err := b.client.Services.BlobContainers.Create(
		ctx,
		b.Dependencies.ResourceGroup.Identifier,
		b.Dependencies.StorageAccount.Identifier,
		b.Identifier,
		storage.BlobContainer{})
	if err != nil {
		return err
	}

	b.Resource = &container
	return nil
}

func (b *BlobContainer) Read(ctx context.Context) error {
	container, err := b.client.Services.BlobContainers.Get(ctx, b.Dependencies.ResourceGroup.Identifier, b.Dependencies.StorageAccount.Identifier, b.Identifier)
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	b.Resource = &container
	return nil
}

func (b *BlobContainer) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (b *BlobContainer) Delete(ctx context.Context) error {
	_, err := b.client.Services.BlobContainers.Delete(ctx, b.Dependencies.ResourceGroup.Identifier, b.Dependencies.StorageAccount.Identifier, b.Identifier)
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return nil
		}
		return err
	}

	b.Resource = nil
	return nil
}

// ConnectionString implements BucketCredentials.
// The method returns the rclone connection string for the specific bucket.
func (b *BlobContainer) ConnectionString(ctx context.Context) (string, error) {
	connection := machine.RcloneConnection{
		Backend:   machine.RcloneBackendAzureBlob,
		Container: b.Dependencies.StorageAccount.Identifier,
		Config: map[string]string{
			"account": b.Dependencies.StorageAccount.Identifier,
			"key":     to.String(b.Dependencies.StorageAccount.Attributes.Value),
		},
	}
	return connection.String(), nil
}

var _ common.StorageCredentials = (*BlobContainer)(nil)
