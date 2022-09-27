package resources

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
)

func NewVirtualNetwork(client *client.Client, identifier common.Identifier, resourceGroup *ResourceGroup) *VirtualNetwork {
	v := &VirtualNetwork{
		client:     client,
		Identifier: identifier.Long(),
	}
	v.Dependencies.ResourceGroup = resourceGroup
	return v
}

type VirtualNetwork struct {
	client       *client.Client
	Identifier   string
	Dependencies struct {
		ResourceGroup *ResourceGroup
	}
	Resource *armnetwork.VirtualNetwork
}

func (v *VirtualNetwork) Create(ctx context.Context) error {
	// Guard against InUseSubnetCannotBeDeleted for existing virtual networks
	if err := v.Read(ctx); err == nil {
		return nil
	}

	poller, err := v.client.Services.VirtualNetworks.BeginCreateOrUpdate(
		ctx,
		v.Dependencies.ResourceGroup.Identifier,
		v.Identifier,
		armnetwork.VirtualNetwork{
			Tags:     v.client.Tags,
			Location: to.Ptr(v.client.Region),
			Properties: &armnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &armnetwork.AddressSpace{
					AddressPrefixes: []*string{to.Ptr("10.0.0.0/8")},
				},
			},
		},
		nil,
	)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return err
	}
	return v.Read(ctx)
}

func (v *VirtualNetwork) Read(ctx context.Context) error {
	response, err := v.client.Services.VirtualNetworks.Get(ctx, v.Dependencies.ResourceGroup.Identifier, v.Identifier, nil)
	if err != nil {
		var e *azcore.ResponseError
		if errors.As(err, &e) && e.RawResponse.StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	v.Resource = &response.VirtualNetwork
	return nil
}

func (v *VirtualNetwork) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (v *VirtualNetwork) Delete(ctx context.Context) error {
	poller, err := v.client.Services.VirtualNetworks.BeginDelete(ctx, v.Dependencies.ResourceGroup.Identifier, v.Identifier, nil)
	if err != nil {
		var e *azcore.ResponseError
		if errors.As(err, &e) && e.RawResponse.StatusCode == 404 {
			return nil
		}
		return err
	}

	if _, err := poller.PollUntilDone(ctx, nil); err != nil {
		return err
	}

	v.Resource = nil
	return nil
}
