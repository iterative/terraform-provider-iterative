package resources

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
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
	Resource *armstorage.BlobContainer
}

func (b *BlobContainer) Create(ctx context.Context) error {
	response, err := b.client.Services.BlobContainers.Create(
		ctx,
		b.Dependencies.ResourceGroup.Identifier,
		b.Dependencies.StorageAccount.Identifier,
		b.Identifier,
		armstorage.BlobContainer{},
		nil,
	)
	if err != nil {
		return err
	}

	b.Resource = &response.BlobContainer
	return nil
}

func (b *BlobContainer) Read(ctx context.Context) error {
	response, err := b.client.Services.BlobContainers.Get(ctx, b.Dependencies.ResourceGroup.Identifier, b.Dependencies.StorageAccount.Identifier, b.Identifier, nil)
	if err != nil {
		var e *azcore.ResponseError
		if errors.As(err, &e) && e.RawResponse.StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	b.Resource = &response.BlobContainer
	return nil
}

func (b *BlobContainer) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (b *BlobContainer) Delete(ctx context.Context) error {
	_, err := b.client.Services.BlobContainers.Delete(ctx, b.Dependencies.ResourceGroup.Identifier, b.Dependencies.StorageAccount.Identifier, b.Identifier, nil)
	if err != nil {
		var e *azcore.ResponseError
		if errors.As(err, &e) && e.RawResponse.StatusCode == 404 {
			return nil
		}
		return err
	}

	b.Resource = nil
	return nil
}
