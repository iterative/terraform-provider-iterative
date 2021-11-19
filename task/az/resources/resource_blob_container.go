package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-04-01/storage"
	"github.com/Azure/go-autorest/autorest"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/universal"
)

func NewBlobContainer(client *client.Client, identifier universal.Identifier, resourceGroup *ResourceGroup, storageAccount *StorageAccount) *BlobContainer {
	b := new(BlobContainer)
	b.Client = client
	b.Identifier = identifier.Short()
	b.Dependencies.ResourceGroup = resourceGroup
	b.Dependencies.StorageAccount = storageAccount
	return b
}

type BlobContainer struct {
	Client       *client.Client
	Identifier   string
	Dependencies struct {
		*ResourceGroup
		*StorageAccount
	}
	Resource *storage.BlobContainer
}

func (b *BlobContainer) Create(ctx context.Context) error {
	container, err := b.Client.Services.BlobContainers.Create(
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
	container, err := b.Client.Services.BlobContainers.Get(ctx, b.Dependencies.ResourceGroup.Identifier, b.Dependencies.StorageAccount.Identifier, b.Identifier)
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return universal.NotFoundError
		}
		return err
	}

	b.Resource = &container
	return nil
}

func (b *BlobContainer) Update(ctx context.Context) error {
	return universal.NotImplementedError
}

func (b *BlobContainer) Delete(ctx context.Context) error {
	_, err := b.Client.Services.BlobContainers.Delete(ctx, b.Dependencies.ResourceGroup.Identifier, b.Dependencies.StorageAccount.Identifier, b.Identifier)
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return nil
		}
		return err
	}

	b.Resource = nil
	return nil
}
