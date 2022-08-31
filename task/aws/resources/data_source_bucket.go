package resources

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/common"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// NewExistingS3Bucket returns a new data source refering to a pre-allocated
// S3 bucket.
func NewExistingS3Bucket(client *client.Client, id string, path string) *ExistingS3Bucket {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return &ExistingS3Bucket{
		client: client,

		id:   id,
		path: path,
	}
}

// ExistingS3Bucket identifies an existing S3 bucket.
type ExistingS3Bucket struct {
	client *client.Client

	resource *types.Bucket
	id       string
	path     string
}

// Read verifies the specified S3 bucket is accessible.
func (b *ExistingS3Bucket) Read(ctx context.Context) error {
	input := s3.HeadBucketInput{
		Bucket: aws.String(b.id),
	}
	if _, err := b.client.Services.S3.HeadBucket(ctx, &input); err != nil {
		if errorCodeIs(err, errNotFound) {
			return common.NotFoundError
		}
		return err
	}
	b.resource = &types.Bucket{Name: aws.String(b.id)}
	return nil
}

// ConnectionString implements common.StorageCredentials.
// The method returns the rclone connection string for the specific bucket.
func (b *ExistingS3Bucket) ConnectionString(ctx context.Context) (string, error) {
	credentials, err := b.client.Config.Credentials.Retrieve(ctx)
	if err != nil {
		return "", err
	}

	connectionString := fmt.Sprintf(
		":s3,provider=AWS,region=%s,access_key_id=%s,secret_access_key=%s,session_token=%s:%s/%s",
		b.client.Region,
		credentials.AccessKeyID,
		credentials.SecretAccessKey,
		credentials.SessionToken,
		b.id,
		b.path)
	return connectionString, nil
}

// build-time check to ensure Bucket implements BucketCredentials.
var _ common.StorageCredentials = (*ExistingS3Bucket)(nil)
