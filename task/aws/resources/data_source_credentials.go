package resources

import (
	"context"

	"github.com/iterative/terraform-provider-iterative/task/aws/client"
	"github.com/iterative/terraform-provider-iterative/task/common"
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
	credentials, err := c.client.Config.Credentials.Retrieve(ctx)
	if err != nil {
		return err
	}

	bucketConnStr, err := c.Dependencies.Bucket.ConnectionString(ctx)
	if err != nil {
		return err
	}

	c.Resource = map[string]string{
		"AWS_ACCESS_KEY_ID":       credentials.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY":   credentials.SecretAccessKey,
		"AWS_SESSION_TOKEN":       credentials.SessionToken,
		"RCLONE_REMOTE":           bucketConnStr,
		"TPI_TASK_CLOUD_PROVIDER": string(c.client.Cloud.Provider),
		"TPI_TASK_CLOUD_REGION":   c.client.Region,
		"TPI_TASK_IDENTIFIER":     c.Identifier,
	}

	return nil
}
