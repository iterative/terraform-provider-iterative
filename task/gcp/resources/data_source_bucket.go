package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/storage/v1"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/gcp/client"
)

// NewExistingBucket creates a new data source referring to a pre-allocated GCP storage bucket.
func NewExistingBucket(client *client.Client, id string, path string) *ExistingBucket {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return &ExistingBucket{
		client: client,

		id:   id,
		path: path,
	}
}

// ExistingBucket identifies a pre-allocated storage bucket.
type ExistingBucket struct {
	client *client.Client

	resource *storage.Bucket
	id       string
	path     string
}

// Read verifies the specified storage bucket exists and is accessible.
func (b *ExistingBucket) Read(ctx context.Context) error {
	bucket, err := b.client.Services.Storage.Buckets.Get(b.id).Do()
	if err != nil {
		var e *googleapi.Error
		if errors.As(err, &e) && e.Code == 404 {
			return common.NotFoundError
		}
		return err
	}

	b.resource = bucket
	return nil
}

// ConnectionString implements common.StorageCredentials.
// The method returns the rclone connection string for the specific bucket.
func (b *ExistingBucket) ConnectionString(ctx context.Context) (string, error) {
	if len(b.client.Credentials.JSON) == 0 {
		return "", errors.New("unable to find credentials JSON string")
	}
	credentials := string(b.client.Credentials.JSON)

	connStr := fmt.Sprintf(
		":googlecloudstorage,service_account_credentials='%s':%s",
		credentials,
		b.id,
	)

	return connStr, nil
}

var _ common.StorageCredentials = (*ExistingBucket)(nil)
