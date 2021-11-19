package resources

import (
	"context"
	"fmt"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/common"
)

func NewCredentials(client *client.Client, identifier common.Identifier, bucket *Bucket) *Credentials {
	c := new(Credentials)
	c.Client = client
	c.Identifier = identifier.Long()
	c.Dependencies.Bucket = bucket
	return c
}

type Credentials struct {
	Client       *client.Client
	Identifier   string
	Dependencies struct {
		*Bucket
	}
	Resource *map[string]string
}

func (c *Credentials) Read(ctx context.Context) error {
	credentials, err := c.Client.Config.Credentials.Retrieve(ctx)
	if err != nil {
		return err
	}

	connectionString := fmt.Sprintf(
		":s3,provider=AWS,region=%s,access_key_id=%s,secret_access_key=%s,session_token=%s:%s",
		c.Client.Region,
		credentials.AccessKeyID,
		credentials.SecretAccessKey,
		credentials.SessionToken,
		c.Dependencies.Bucket.Identifier,
	)

	c.Resource = &map[string]string{
		"AWS_ACCESS_KEY_ID":       credentials.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY":   credentials.SecretAccessKey,
		"AWS_SESSION_TOKEN":       credentials.SessionToken,
		"RCLONE_REMOTE":           connectionString,
		"TPI_TASK_CLOUD_PROVIDER": string(c.Client.Cloud.Provider),
		"TPI_TASK_CLOUD_REGION":   c.Client.Region,
		"TPI_TASK_IDENTIFIER":     c.Identifier,
	}

	return nil
}
