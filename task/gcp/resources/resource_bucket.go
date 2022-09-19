package resources

import (
	"context"
	"errors"

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
	return &Bucket{
		client:     client,
		Identifier: identifier.Long(),
	}
}

type Bucket struct {
	client     *client.Client
	Identifier string
	Resource   *storage.Bucket
}

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

func (b *Bucket) Update(ctx context.Context) error {
	return common.NotImplementedError
}

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
