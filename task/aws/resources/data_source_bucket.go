package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/iterative/terraform-provider-iterative/task/common"
	"github.com/iterative/terraform-provider-iterative/task/common/machine"
)

// NewExistingS3Bucket returns a new data source refering to a pre-allocated
// S3 bucket.
func NewExistingS3Bucket(credentials aws.Credentials, storageParams common.RemoteStorage) *ExistingS3Bucket {
	return &ExistingS3Bucket{
		credentials: credentials,
		params:      storageParams,
	}
}

// ExistingS3Bucket identifies an existing S3 bucket.
type ExistingS3Bucket struct {
	credentials aws.Credentials

	params common.RemoteStorage
}

// Read verifies the specified S3 bucket is accessible.
func (b *ExistingS3Bucket) Read(ctx context.Context) error {
	err := machine.CheckStorage(ctx, b.connection())
	if err != nil {
		return fmt.Errorf("failed to verify existing s3 bucket: %w", err)
	}
	return nil
}

func (b *ExistingS3Bucket) connection() machine.RcloneConnection {
	region := b.params.Config["region"]
	return machine.RcloneConnection{
		Backend:   machine.RcloneBackendS3,
		Container: b.params.Container,
		Path:      b.params.Path,
		Config: map[string]string{
			"provider":          "AWS",
			"region":            region,
			"access_key_id":     b.credentials.AccessKeyID,
			"secret_access_key": b.credentials.SecretAccessKey,
			"session_token":     b.credentials.SessionToken,
		},
	}
}

// ConnectionString implements common.StorageCredentials.
// The method returns the rclone connection string for the specific bucket.
func (b *ExistingS3Bucket) ConnectionString(ctx context.Context) (string, error) {
	connection := b.connection()
	return connection.String(), nil
}

// build-time check to ensure Bucket implements BucketCredentials.
var _ common.StorageCredentials = (*ExistingS3Bucket)(nil)
