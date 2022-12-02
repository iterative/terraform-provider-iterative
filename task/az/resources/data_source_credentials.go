package resources

import (
	"context"
	"errors"

	"github.com/iterative/terraform-provider-iterative/task/az/client"
	"github.com/iterative/terraform-provider-iterative/task/common"
)

func NewCredentials(client *client.Client, identifier common.Identifier, resourceGroup *ResourceGroup, blobContainer common.StorageCredentials) *Credentials {
	c := &Credentials{
		client:     client,
		Identifier: identifier.Long(),
	}
	c.Dependencies.ResourceGroup = resourceGroup
	c.Dependencies.BlobContainer = blobContainer
	return c
}

type Credentials struct {
	client       *client.Client
	Identifier   string
	Dependencies struct {
		ResourceGroup  *ResourceGroup
		StorageAccount *StorageAccount
		BlobContainer  common.StorageCredentials
	}
	Resource map[string]string
}

func (c *Credentials) Read(ctx context.Context) error {
	credentials := c.client.Cloud.Credentials.AZCredentials

	if len(credentials.ClientSecret) == 0 {
		return errors.New("unable to find client secret")
	}

	connectionString, err := c.Dependencies.BlobContainer.ConnectionString(ctx)
	if err != nil {
		return err
	}

	c.Resource = map[string]string{
		"AZURE_CLIENT_ID":         credentials.ClientID,
		"AZURE_CLIENT_SECRET":     credentials.ClientSecret,
		"AZURE_SUBSCRIPTION_ID":   credentials.SubscriptionID,
		"AZURE_TENANT_ID":         credentials.TenantID,
		"RCLONE_REMOTE":           connectionString,
		"TPI_TASK_CLOUD_PROVIDER": string(c.client.Cloud.Provider),
		"TPI_TASK_CLOUD_REGION":   c.client.Region,
		"TPI_TASK_IDENTIFIER":     c.Identifier,
	}

	return nil
}
