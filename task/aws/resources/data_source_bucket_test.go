package resources_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/require"

	"terraform-provider-iterative/task/aws/resources"
	"terraform-provider-iterative/task/common"
)

func TestExistingBucketConnectionString(t *testing.T) {
	ctx := context.Background()
	creds := aws.Credentials{
		AccessKeyID:     "access-key-id",
		SecretAccessKey: "secret-access-key",
		SessionToken:    "session-token",
	}
	b := resources.NewExistingS3Bucket(creds, common.RemoteStorage{
		Container: "pre-created-bucket",
		Config:    map[string]string{"region": "us-east-1"},
		Path:      "subdirectory"})
	connStr, err := b.ConnectionString(ctx)
	require.NoError(t, err)
	require.Equal(t, ":s3,access_key_id='access-key-id',provider='AWS',region='us-east-1',secret_access_key='secret-access-key',session_token='session-token':pre-created-bucket/subdirectory", connStr)
}
