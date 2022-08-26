package resources

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"terraform-provider-iterative/task/common"
)

// NewExistingS3Bucket returns a new data source refering to a pre-allocated
// S3 bucket.
func NewExistingS3Bucket(client S3Client, credentials aws.Credentials, id string, region string, path string) *ExistingS3Bucket {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return &ExistingS3Bucket{
		client:      client,
		credentials: credentials,
		region:      region,
		id:          id,
		path:        path,
	}
}

// ExistingS3Bucket identifies an existing S3 bucket.
type ExistingS3Bucket struct {
	client      S3Client
	credentials aws.Credentials

	id     string
	region string
	path   string
}

// Read verifies the specified S3 bucket is accessible.
func (b *ExistingS3Bucket) Read(ctx context.Context) error {
	input := s3.HeadBucketInput{
		Bucket: aws.String(b.id),
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
	containerPath := path.Join(b.id, b.path)
	connectionString := fmt.Sprintf(
		":s3,provider=AWS,region=%s,access_key_id=%s,secret_access_key=%s,session_token=%s:%s/%s",
		b.region,
		b.credentials.AccessKeyID,
		b.credentials.SecretAccessKey,
		b.credentials.SessionToken,
		b.id,
		containerPath)
	return connectionString, nil
}

// build-time check to ensure Bucket implements BucketCredentials.
var _ common.StorageCredentials = (*ExistingS3Bucket)(nil)

// S3Client defines the functions of the AWS S3 API used.
type S3Client interface {
	HeadBucket(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}
