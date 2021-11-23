package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
)

func NewSubnet(client *client.Client, identifier common.Identifier, resourceGroup *ResourceGroup, virtualNetwork *VirtualNetwork, securityGroup *SecurityGroup) *Subnet {
	s := new(Subnet)
	s.Client = client
	s.Identifier = identifier.Long()
	s.Dependencies.ResourceGroup = resourceGroup
	s.Dependencies.VirtualNetwork = virtualNetwork
	s.Dependencies.SecurityGroup = securityGroup
	return s
}

type Subnet struct {
	Client       *client.Client
	Identifier   string
	Dependencies struct {
		*ResourceGroup
		*VirtualNetwork
		*SecurityGroup
	}
	Resource *network.Subnet
}

func (s *Subnet) Create(ctx context.Context) error {
	subnetCreateFuture, err := s.Client.Services.Subnets.CreateOrUpdate(
		ctx,
		s.Dependencies.ResourceGroup.Identifier,
		s.Dependencies.VirtualNetwork.Identifier,
		s.Identifier,
		network.Subnet{
			SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
				AddressPrefix:        to.StringPtr("10.0.0.0/16"),
				NetworkSecurityGroup: s.Dependencies.SecurityGroup.Resource,
			},
		})
	if err != nil {
		return err
	}

	if err := subnetCreateFuture.WaitForCompletionRef(ctx, s.Client.Services.Subnets.Client); err != nil {
		return err
	}

	return s.Read(ctx)
}

func (s *Subnet) Read(ctx context.Context) error {
	subnet, err := s.Client.Services.Subnets.Get(ctx, s.Dependencies.ResourceGroup.Identifier, s.Dependencies.VirtualNetwork.Identifier, s.Identifier, "")
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	s.Resource = &subnet
	return nil
}

func (s *Subnet) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (s *Subnet) Delete(ctx context.Context) error {
	subnetDeleteFuture, err := s.Client.Services.Subnets.Delete(ctx, s.Dependencies.ResourceGroup.Identifier, s.Dependencies.VirtualNetwork.Identifier, s.Identifier)
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return nil
		}
		return err
	}

	err = subnetDeleteFuture.WaitForCompletionRef(ctx, s.Client.Services.Subnets.Client)
	s.Resource = nil
	return err
}
