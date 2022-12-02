package resources

import (
	"context"
	"errors"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"github.com/iterative/terraform-provider-iterative/task/common"
	"github.com/iterative/terraform-provider-iterative/task/gcp/client"
)

func NewDefaultNetwork(client *client.Client) *DefaultNetwork {
	return &DefaultNetwork{
		client: client,
	}
}

type DefaultNetwork struct {
	client   *client.Client
	Resource *compute.Network
}

func (d *DefaultNetwork) Read(ctx context.Context) error {
	network, err := d.client.Services.Compute.Networks.Get(d.client.Credentials.ProjectID, "default").Do()
	if err != nil {
		var e *googleapi.Error
		if errors.As(err, &e) && e.Code == 404 {
			return common.NotFoundError
		}
		return err
	}

	d.Resource = network
	return nil
}
