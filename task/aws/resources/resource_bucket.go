package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/common"
)

const (
	errNoSuchBucket            = "NoSuchBucket"
	errNotFound                = "NotFound"
	errBucketAlreadyOwnedByYou = "BucketAlreadyOwnedByYou"
)

func ListBuckets(ctx context.Context, client *client.Client) ([]common.Identifier, error) {
	output, err := client.Services.S3.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	ids := []common.Identifier{}
	for _, b := range output.Buckets {
		if id, err := common.ParseIdentifier(*b.Name); err == nil {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

func NewBucket(client *client.Client, identifier common.Identifier) *Bucket {
	b := new(Bucket)
	b.Client = client
	b.Identifier = identifier.Long()
	return b
}

type Bucket struct {
	Client     *client.Client
	Identifier string
	Resource   *types.Bucket
}

func (b *Bucket) Create(ctx context.Context) error {
	createInput := s3.CreateBucketInput{
		Bucket: aws.String(b.Identifier),
	}

	if b.Client.Region != "us-east-1" {
		createInput.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(b.Client.Region),
		}
	}

	if _, err := b.Client.Services.S3.CreateBucket(ctx, &createInput); err != nil {
		if errorCodeIs(err, errBucketAlreadyOwnedByYou) {
			return b.Read(ctx)
		}
		return err
	}

	waitInput := s3.HeadBucketInput{
		Bucket: aws.String(b.Identifier),
	}

	if err := s3.NewBucketExistsWaiter(b.Client.Services.S3).Wait(ctx, &waitInput, b.Client.Cloud.Timeouts.Create); err != nil {
		return err
	}

	return b.Read(ctx)
}

func (b *Bucket) Read(ctx context.Context) error {
	input := s3.HeadBucketInput{
		Bucket: aws.String(b.Identifier),
	}

	if _, err := b.Client.Services.S3.HeadBucket(ctx, &input); err != nil {
		if errorCodeIs(err, errNotFound) {
			return common.NotFoundError
		}
		return err
	}

	b.Resource = &types.Bucket{Name: aws.String(b.Identifier)}
	return nil
}

func (b *Bucket) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (b *Bucket) Delete(ctx context.Context) error {
	listInput := s3.ListObjectsV2Input{
		Bucket: aws.String(b.Identifier),
	}

	for paginator := s3.NewListObjectsV2Paginator(b.Client.Services.S3, &listInput); paginator.HasMorePages(); {
		page, err := paginator.NextPage(ctx)
		if errorCodeIs(err, errNoSuchBucket) {
			b.Resource = nil
			return nil
		}
		if err != nil {
			return err
		}

		if len(page.Contents) == 0 {
			break
		}

		var objects []types.ObjectIdentifier
		for _, object := range page.Contents {
			objects = append(objects, types.ObjectIdentifier{
				Key: object.Key,
			})
		}

		input := s3.DeleteObjectsInput{
			Bucket: aws.String(b.Identifier),
			Delete: &types.Delete{
				Objects: objects,
			},
		}

		if _, err = b.Client.Services.S3.DeleteObjects(ctx, &input); err != nil {
			return err
		}
	}

	deleteInput := s3.DeleteBucketInput{
		Bucket: aws.String(b.Identifier),
	}

	_, err := b.Client.Services.S3.DeleteBucket(ctx, &deleteInput)
	if errorCodeIs(err, errNoSuchBucket) {
		b.Resource = nil
		return nil
	}
	if err != nil {
		return err
	}

	b.Resource = nil
	return nil
}

// ConnectionString implements BucketCredentials.
// The method returns the rclone connection string for the specific bucket.
func (b *Bucket) ConnectionString(ctx context.Context) (string, error) {
	credentials, err := b.Client.Config.Credentials.Retrieve(ctx)
	if err != nil {
		return "", err
	}

	connectionString := fmt.Sprintf(
		":s3,provider=AWS,region=%s,access_key_id=%s,secret_access_key=%s,session_token=%s:%s",
		b.Client.Region,
		credentials.AccessKeyID,
		credentials.SecretAccessKey,
		credentials.SessionToken,
		b.Identifier)
	return connectionString, nil
}

// errorCodeIs checks if the provided error is an AWS API error
// and its error code matches the supplied value.
func errorCodeIs(err error, code string) bool {
	var e smithy.APIError
	if errors.As(err, &e) {
		return e.ErrorCode() == code
	}
	return false
}

// build-time check to ensure Bucket implements BucketCredentials.
var _ common.StorageCredentials = (*Bucket)(nil)
