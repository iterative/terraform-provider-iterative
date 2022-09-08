package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"terraform-provider-iterative/task/common"
)

// NewExistingS3Bucket returns a new data source refering to a pre-allocated
// S3 bucket.
func NewExistingS3Bucket(client S3Client, credentials aws.Credentials, storageParams common.RemoteStorage) *ExistingS3Bucket {
	return &ExistingS3Bucket{
		client:      client,
		credentials: credentials,
		params:      storageParams,
	}
}

// ExistingS3Bucket identifies an existing S3 bucket.
type ExistingS3Bucket struct {
	client      S3Client
	credentials aws.Credentials

	params common.RemoteStorage
}

// Read verifies the specified S3 bucket is accessible.
func (b *ExistingS3Bucket) Read(ctx context.Context) error {
	input := s3.HeadBucketInput{
		Bucket: aws.String(b.params.Container),
	}
	if _, err := b.client.HeadBucket(ctx, &input); err != nil {
		if errorCodeIs(err, errNotFound) {
			return common.NotFoundError
		}
		return err
	}
	return nil
}

// ConnectionString implements common.StorageCredentials.
// The method returns the rclone connection string for the specific bucket.
func (b *ExistingS3Bucket) ConnectionString(ctx context.Context) (string, error) {
	region := b.params.Config["region"]
	connectionString := fmt.Sprintf(
		":s3,provider=AWS,region=%s,access_key_id=%s,secret_access_key=%s,session_token=%s:%s/%s",
		region,
		b.credentials.AccessKeyID,
		b.credentials.SecretAccessKey,
		b.credentials.SessionToken,
		b.params.Container,
		strings.TrimPrefix(b.params.Path, "/"))
	return connectionString, nil
}

// build-time check to ensure Bucket implements BucketCredentials.
var _ common.StorageCredentials = (*ExistingS3Bucket)(nil)

// S3Client defines the functions of the AWS S3 API used.
type S3Client interface {
	HeadBucket(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}
