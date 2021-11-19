package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
)

func NewVirtualNetwork(client *client.Client, identifier common.Identifier, resourceGroup *ResourceGroup) *VirtualNetwork {
	v := new(VirtualNetwork)
	v.Client = client
	v.Identifier = identifier.Long()
	v.Dependencies.ResourceGroup = resourceGroup
	return v
}

type VirtualNetwork struct {
	Client       *client.Client
	Identifier   string
	Dependencies struct {
		*ResourceGroup
	}
	Resource *network.VirtualNetwork
}

func (v *VirtualNetwork) Create(ctx context.Context) error {
	// Guard against InUseSubnetCannotBeDeleted for existing virtual networks
	if err := v.Read(ctx); err == nil {
		return nil
	}

	virtualNetworkCreateFuture, err := v.Client.Services.VirtualNetworks.CreateOrUpdate(
		ctx,
		v.Dependencies.ResourceGroup.Identifier,
		v.Identifier,
		network.VirtualNetwork{
			Tags:     v.Client.Tags,
			Location: to.StringPtr(v.Client.Region),
			VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
				AddressSpace: &network.AddressSpace{
					AddressPrefixes: &[]string{"10.0.0.0/8"},
				},
			},
		})
	if err != nil {
		return err
	}
	err = virtualNetworkCreateFuture.WaitForCompletionRef(ctx, v.Client.Services.VirtualNetworks.Client)
	if err != nil {
		return err
	}
	return v.Read(ctx)
}

func (v *VirtualNetwork) Read(ctx context.Context) error {
	virtualNetwork, err := v.Client.Services.VirtualNetworks.Get(ctx, v.Dependencies.ResourceGroup.Identifier, v.Identifier, "")
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	v.Resource = &virtualNetwork
	return nil
}

func (v *VirtualNetwork) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (v *VirtualNetwork) Delete(ctx context.Context) error {
	future, err := v.Client.Services.VirtualNetworks.Delete(ctx, v.Dependencies.ResourceGroup.Identifier, v.Identifier)
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return nil
		}
		return err
	}

	if err := future.WaitForCompletionRef(ctx, v.Client.Services.VirtualNetworks.Client); err != nil {
		return err
	}

	v.Resource = nil
	return nil
}
