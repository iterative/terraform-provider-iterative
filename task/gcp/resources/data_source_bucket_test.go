package resources_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iterative/terraform-provider-iterative/task/common"
	"github.com/iterative/terraform-provider-iterative/task/gcp/resources"
)

func TestExistingBucketConnectionString(t *testing.T) {
	ctx := context.Background()
	creds := "gcp-credentials-json"
	b := resources.NewExistingBucket(creds, common.RemoteStorage{
		Container: "pre-created-bucket",
		Path:      "subdirectory"})
	connStr, err := b.ConnectionString(ctx)
	require.NoError(t, err)
	require.Equal(t, ":googlecloudstorage,service_account_credentials='gcp-credentials-json':pre-created-bucket/subdirectory", connStr)
}
