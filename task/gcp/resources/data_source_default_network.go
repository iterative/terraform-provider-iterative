package resources

import (
	"context"
	"errors"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"terraform-provider-iterative/task/gcp/client"
	"terraform-provider-iterative/task/universal"
)

func NewDefaultNetwork(client *client.Client) *DefaultNetwork {
	d := new(DefaultNetwork)
	d.Client = client
	return d
}

type DefaultNetwork struct {
	Client   *client.Client
	Resource *compute.Network
}

func (d *DefaultNetwork) Read(ctx context.Context) error {
	network, err := d.Client.Services.Compute.Networks.Get(d.Client.Credentials.ProjectID, "default").Do()
	if err != nil {
		var e *googleapi.Error
		if errors.As(err, &e) && e.Code == 404 {
			return universal.NotFoundError
		}
		return err
	}

	d.Resource = network
	return nil
}
