package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
)

func NewCredentials(client *client.Client, identifier common.Identifier, resourceGroup *ResourceGroup, storageAccount *StorageAccount, blobContainer *BlobContainer) *Credentials {
	c := new(Credentials)
	c.Client = client
	c.Identifier = identifier.Long()
	c.Dependencies.ResourceGroup = resourceGroup
	c.Dependencies.StorageAccount = storageAccount
	c.Dependencies.BlobContainer = blobContainer
	return c
}

type Credentials struct {
	Client       *client.Client
	Identifier   string
	Dependencies struct {
		*ResourceGroup
		*StorageAccount
		*BlobContainer
	}
	Resource *map[string]string
}

func (c *Credentials) Read(ctx context.Context) error {
	connectionString := fmt.Sprintf(
		":azureblob,account='%s',key='%s':%s",
		c.Dependencies.StorageAccount.Identifier,
		to.String(c.Dependencies.StorageAccount.Attributes.Value),
		c.Dependencies.BlobContainer.Identifier,
	)

	credentials, err := c.Client.Settings.GetClientCredentials()
	if err != nil {
		return err
	}

	if len(credentials.ClientSecret) == 0 {
		return errors.New("unable to find client secret")
	}

	subscriptionID := c.Client.Settings.GetSubscriptionID()

	c.Resource = &map[string]string{
		"AZURE_CLIENT_ID":         credentials.ClientID,
		"AZURE_CLIENT_SECRET":     credentials.ClientSecret,
		"AZURE_SUBSCRIPTION_ID":   subscriptionID,
		"AZURE_TENANT_ID":         credentials.TenantID,
		"RCLONE_REMOTE":           connectionString,
		"TPI_TASK_CLOUD_PROVIDER": string(c.Client.Cloud.Provider),
		"TPI_TASK_CLOUD_REGION":   c.Client.Region,
		"TPI_TASK_IDENTIFIER":     c.Identifier,
	}

	return nil
}
