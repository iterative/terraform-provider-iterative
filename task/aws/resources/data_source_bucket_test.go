package resources_test

import (
	"context"
	"testing"

	"terraform-provider-iterative/task/aws/resources"
	"terraform-provider-iterative/task/aws/resources/mocks"
	"terraform-provider-iterative/task/common"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestExistingBucketConnectionString(t *testing.T) {
	ctx := context.Background()
	creds := aws.Credentials{
		AccessKeyID:     "access-key-id",
		SecretAccessKey: "secret-access-key",
		SessionToken:    "session-token",
	}
	b := resources.NewExistingS3Bucket(nil, creds, "pre-created-bucket", "us-east-1", "subdirectory")
	connStr, err := b.ConnectionString(ctx)
	require.NoError(t, err)
	require.Equal(t, connStr, ":s3,provider=AWS,region=us-east-1,access_key_id=access-key-id,secret_access_key=secret-access-key,session_token=session-token:pre-created-bucket/subdirectory")
}

func TestExistingBucketRead(t *testing.T) {
	ctx := context.Background()
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	s3Cl := mocks.NewMockS3Client(ctl)
	s3Cl.EXPECT().HeadBucket(gomock.Any(), &s3.HeadBucketInput{Bucket: aws.String("bucket-id")}).Return(nil, nil)
	b := resources.NewExistingS3Bucket(s3Cl, aws.Credentials{}, "bucket-id", "us-east-1", "subdirectory")
	err := b.Read(ctx)
	require.NoError(t, err)
}

// TestExistingBucketReadNotFound tests the case where the s3 client indicates that the bucket could not be
// found.
func TestExistingBucketReadNotFound(t *testing.T) {
	ctx := context.Background()
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	s3Cl := mocks.NewMockS3Client(ctl)

	s3Cl.EXPECT().
		HeadBucket(gomock.Any(), &s3.HeadBucketInput{Bucket: aws.String("bucket-id")}).
		Return(nil, &smithy.GenericAPIError{Code: "NotFound"})
	b := resources.NewExistingS3Bucket(s3Cl, aws.Credentials{}, "bucket-id", "us-east-1", "subdirectory")
	err := b.Read(ctx)
	require.ErrorIs(t, err, common.NotFoundError)
}
