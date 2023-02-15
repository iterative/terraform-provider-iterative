package resources

import (
	"context"
	"errors"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/gcp/client"
)

func NewCredentials(client *client.Client, identifier common.Identifier, bucket common.StorageCredentials) *Credentials {
	c := &Credentials{
		client:     client,
		Identifier: identifier.Long(),
	}
	c.Dependencies.Bucket = bucket
	return c
}

type Credentials struct {
	client       *client.Client
	Identifier   string
	Dependencies struct {
		Bucket common.StorageCredentials
	}
	Resource map[string]string
}

func (c *Credentials) Read(ctx context.Context) error {
	if len(c.client.Credentials.JSON) == 0 {
		return errors.New("unable to find credentials JSON string")
	}
	credentials := string(c.client.Credentials.JSON)

	connectionString, err := c.Dependencies.Bucket.ConnectionString(ctx)
	if err != nil {
		return err
	}
	c.Resource = map[string]string{
		"GOOGLE_APPLICATION_CREDENTIALS_DATA": credentials,
		"RCLONE_REMOTE":                       connectionString,
		"TPI_TASK_CLOUD_PROVIDER":             string(c.client.Cloud.Provider),
		"TPI_TASK_CLOUD_REGION":               c.client.Region,
		"TPI_TASK_IDENTIFIER":                 c.Identifier,
	}

	return nil
}
