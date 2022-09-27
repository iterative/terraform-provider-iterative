package resources

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
)

func NewStorageAccount(client *client.Client, identifier common.Identifier, resourceGroup *ResourceGroup) *StorageAccount {
	s := &StorageAccount{
		client:     client,
		Identifier: identifier.Short(),
	}
	s.Dependencies.ResourceGroup = resourceGroup
	return s
}

type StorageAccount struct {
	client       *client.Client
	Identifier   string
	Attributes   *armstorage.AccountKey
	Dependencies struct {
		ResourceGroup *ResourceGroup
	}
	Resource *armstorage.Account
}

func (s *StorageAccount) Create(ctx context.Context) error {
	poller, err := s.client.Services.StorageAccounts.BeginCreate(
		ctx,
		s.Dependencies.ResourceGroup.Identifier,
		s.Identifier,
		armstorage.AccountCreateParameters{
			SKU: &armstorage.SKU{
				Name: to.Ptr(armstorage.SKUNameStandardLRS),
				Tier: to.Ptr(armstorage.SKUTierStandard),
			},
			Kind:     to.Ptr(armstorage.KindBlobStorage),
			Location: to.Ptr(s.client.Region),
			Tags:     s.client.Tags,
			Properties: &armstorage.AccountPropertiesCreateParameters{
				AccessTier: to.Ptr(armstorage.AccessTierHot),
			},
		},
		nil,
	)
	if err != nil {
		return err
	}

	if _, err := poller.PollUntilDone(ctx, nil); err != nil {
		return err
	}
	return s.Read(ctx)
}

func (s *StorageAccount) Read(ctx context.Context) error {
	response, err := s.client.Services.StorageAccounts.GetProperties(ctx, s.Dependencies.ResourceGroup.Identifier, s.Identifier, nil)
	if err != nil {
		var e *azcore.ResponseError
		if errors.As(err, &e) && e.RawResponse.StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	keys, err := s.client.Services.StorageAccounts.ListKeys(ctx, s.Dependencies.ResourceGroup.Identifier, s.Identifier, nil)
	if err != nil {
		return err
	}

	if keys.Keys == nil {
		return errors.New("storage account keys not found")
	}

	for _, key := range keys.Keys {
		actual := strings.ToUpper(string(*key.Permissions))
		expected := strings.ToUpper(string(armstorage.KeyPermissionFull))
		if actual == expected {
			s.Attributes = key
		}
	}

	if s.Attributes == nil {
		return errors.New("storage account read+write key not found")
	}

	s.Resource = &response.Account
	return nil
}

func (s *StorageAccount) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (s *StorageAccount) Delete(ctx context.Context) error {
	_, err := s.client.Services.StorageAccounts.Delete(ctx, s.Dependencies.ResourceGroup.Identifier, s.Identifier, nil)
	if err != nil {
		var e *azcore.ResponseError
		if errors.As(err, &e) && e.RawResponse.StatusCode == 404 {
			return nil
		}
		return err
	}

	s.Resource = nil
	return nil
}
