package resources

import (
	"context"
	"errors"
	"log"
	"time"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/storage/v1"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/gcp/client"
)

func NewBucket(client *client.Client, identifier common.Identifier) *Bucket {
	b := new(Bucket)
	b.Client = client
	b.Identifier = identifier.Long()
	return b
}

type Bucket struct {
	Client     *client.Client
	Identifier string
	Resource   *storage.Bucket
}

func (b *Bucket) Create(ctx context.Context) error {
	bucket, err := b.Client.Services.Storage.Buckets.Insert(b.Client.Credentials.ProjectID, &storage.Bucket{
		Name:     b.Identifier,
		Location: b.Client.Region[:len(b.Client.Region)-2], // remove zone suffix (e.g. `{region}-a` -> `{region}`)
		Labels:   b.Client.Tags,
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
	bucket, err := b.Client.Services.Storage.Buckets.Get(b.Identifier).Do()
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
	for i, err := 0, b.Client.Services.Storage.Buckets.Delete(b.Identifier).Do(); b.Read(ctx) != common.NotFoundError; i++ {
	    log.Println("[DEBUG] Deleting Bucket...")
		var e *googleapi.Error
		if !errors.As(err, &e) || e.Code != 409 {
			return err
		} else if i > 30 {
			return errors.New("timed out waiting for bucket to be deleted")
		}
		time.Sleep(10 * time.Second)
	}

	b.Resource = nil
	return nil
}
