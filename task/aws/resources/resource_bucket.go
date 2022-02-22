package resources

import (
	"context"
	"errors"

	"github.com/aws/smithy-go"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/common"
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
	Resource   *types.Bucket
}

func (b *Bucket) Create(ctx context.Context) error {
	createInput := s3.CreateBucketInput{
		Bucket: aws.String(b.Identifier),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(b.Client.Region),
		},
	}

	if _, err := b.Client.Services.S3.CreateBucket(ctx, &createInput); err != nil {
		var e smithy.APIError
		if errors.As(err, &e) && e.ErrorCode() == "BucketAlreadyOwnedByYou" {
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
		var e smithy.APIError
		if errors.As(err, &e) && e.ErrorCode() == "NotFound" {
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

		if err != nil {
			var e smithy.APIError
			if errors.As(err, &e) && e.ErrorCode() == "NoSuchBucket" {
				b.Resource = nil
				return nil
			}
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
	if err != nil {
		return err
	}

	b.Resource = nil
	return nil
}
