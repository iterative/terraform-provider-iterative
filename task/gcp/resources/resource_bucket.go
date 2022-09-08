package resources

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/storage/v1"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/gcp/client"
)

func ListBuckets(ctx context.Context, client *client.Client) ([]common.Identifier, error) {
	ids := []common.Identifier{}

	page := func(buckets *storage.Buckets) error {
		for _, bucket := range buckets.Items {
			if id, err := common.ParseIdentifier(bucket.Name); err == nil {
				ids = append(ids, id)
			}
		}
		return nil
	}

	if err := client.Services.Storage.Buckets.List(client.Credentials.ProjectID).Pages(ctx, page); err != nil {
		return nil, err
	}

	return ids, nil
}

func NewBucket(client *client.Client, identifier common.Identifier) *Bucket {
	b := new(Bucket)
	b.client = client
	b.Identifier = identifier.Long()
	return b
}

// Bucket is a resource refering to an allocated gcp storage bucket.
type Bucket struct {
	client     *client.Client
	Identifier string
	Resource   *storage.Bucket
}

// Create creates a new gcp storage bucket.
func (b *Bucket) Create(ctx context.Context) error {
	bucket, err := b.client.Services.Storage.Buckets.Insert(b.client.Credentials.ProjectID, &storage.Bucket{
		Name:     b.Identifier,
		Location: b.client.Region[:len(b.client.Region)-2], // remove zone suffix (e.g. `{region}-a` -> `{region}`)
		Labels:   b.client.Tags,
	}).Do()
	if err != nil {
		var e *googleapi.Error
		if errors.As(err, &e) && e.Code == 409 {
			return b.Read(ctx)
		}
		return err
	}

	b.Resource = bucket
	return nil
}

// Read verifies an existing gcp storage bucket.
func (b *Bucket) Read(ctx context.Context) error {
	bucket, err := b.client.Services.Storage.Buckets.Get(b.Identifier).Do()
	if err != nil {
		var e *googleapi.Error
		if errors.As(err, &e) && e.Code == 404 {
			return common.NotFoundError
		}
		return err
	}

	b.Resource = bucket
	return nil
}

// Update implements resource.Update.
// The operation is not implemented for storage buckets.
func (b *Bucket) Update(ctx context.Context) error {
	return common.NotImplementedError
}

// Delete deletes all objects stored in the bucket and destroys
// the storage bucket itself.
func (b *Bucket) Delete(ctx context.Context) error {
	if b.Read(ctx) == common.NotFoundError {
		return nil
	}

	deletePage := func(objects *storage.Objects) error {
		for _, object := range objects.Items {
			if err := b.client.Services.Storage.Objects.Delete(b.Identifier, object.Name).Do(); err != nil {
				return err
			}
		}
		return nil
	}

	if err := b.client.Services.Storage.Objects.List(b.Identifier).Pages(ctx, deletePage); err != nil {
		return err
	}

	if err := b.client.Services.Storage.Buckets.Delete(b.Identifier).Do(); err != nil {
		return err
	}

	b.Resource = nil
	return nil
}

// ConnectionString implements common.StorageCredentials.
// The method returns the rclone connection string for the specific bucket.
func (b *Bucket) ConnectionString(ctx context.Context) (string, error) {
	if len(b.client.Credentials.JSON) == 0 {
		return "", errors.New("unable to find credentials JSON string")
	}
	credentials := string(b.client.Credentials.JSON)

	connStr := fmt.Sprintf(
		":googlecloudstorage,service_account_credentials='%s':%s",
		credentials,
		b.Identifier,
	)

	return connStr, nil
}
