package resources

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
)

func NewStorageAccount(client *client.Client, identifier common.Identifier, resourceGroup *ResourceGroup) *StorageAccount {
	s := new(StorageAccount)
	s.Client = client
	s.Identifier = identifier.Short()
	s.Dependencies.ResourceGroup = resourceGroup
	return s
}

type StorageAccount struct {
	Client       *client.Client
	Identifier   string
	Attributes   *storage.AccountKey
	Dependencies struct {
		*ResourceGroup
	}
	Resource *storage.Account
}

func (s *StorageAccount) Create(ctx context.Context) error {
	future, err := s.Client.Services.StorageAccounts.Create(
		ctx,
		s.Dependencies.ResourceGroup.Identifier,
		s.Identifier,
		storage.AccountCreateParameters{
			Sku: &storage.Sku{
				Name: storage.SkuNameStandardLRS,
				Tier: storage.SkuTierStandard,
			},
			Kind:     storage.KindBlobStorage,
			Location: to.StringPtr(s.Client.Region),
			Tags:     s.Client.Tags,
			AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{
				AccessTier: storage.AccessTierHot,
			},
		})
	if err != nil {
		return err
	}

	if err := future.WaitForCompletionRef(ctx, s.Client.Services.StorageAccounts.Client); err != nil {
		return err
	}
	return s.Read(ctx)
}

func (s *StorageAccount) Read(ctx context.Context) error {
	account, err := s.Client.Services.StorageAccounts.GetProperties(ctx, s.Dependencies.ResourceGroup.Identifier, s.Identifier, "")
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	keys, err := s.Client.Services.StorageAccounts.ListKeys(ctx, s.Dependencies.ResourceGroup.Identifier, s.Identifier, "")
	if err != nil {
		return err
	}

	if keys.Keys == nil {
		return errors.New("storage account keys not found")
	}

	for _, key := range *keys.Keys {
		actual := strings.ToUpper(string(key.Permissions))
		expected := strings.ToUpper(string(storage.KeyPermissionFull))
		if actual == expected {
			s.Attributes = &key
		}
	}

	if s.Attributes == nil {
		return errors.New("storage account read+write key not found")
	}

	s.Resource = &account
	return nil
}

func (s *StorageAccount) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (s *StorageAccount) Delete(ctx context.Context) error {
	_, err := s.Client.Services.StorageAccounts.Delete(ctx, s.Dependencies.ResourceGroup.Identifier, s.Identifier)
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return nil
		}
		return err
	}

	s.Resource = nil
	return nil
}
