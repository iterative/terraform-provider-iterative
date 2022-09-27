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

func NewSubnet(client *client.Client, identifier common.Identifier, resourceGroup *ResourceGroup, virtualNetwork *VirtualNetwork, securityGroup *SecurityGroup) *Subnet {
	s := &Subnet{
		client:     client,
		Identifier: identifier.Long(),
	}
	s.Dependencies.ResourceGroup = resourceGroup
	s.Dependencies.VirtualNetwork = virtualNetwork
	s.Dependencies.SecurityGroup = securityGroup
	return s
}

type Subnet struct {
	client       *client.Client
	Identifier   string
	Dependencies struct {
		ResourceGroup  *ResourceGroup
		VirtualNetwork *VirtualNetwork
		SecurityGroup  *SecurityGroup
	}
	Resource *armnetwork.Subnet
}

func (s *Subnet) Create(ctx context.Context) error {
	poller, err := s.client.Services.Subnets.BeginCreateOrUpdate(
		ctx,
		s.Dependencies.ResourceGroup.Identifier,
		s.Dependencies.VirtualNetwork.Identifier,
		s.Identifier,
		armnetwork.Subnet{
			Properties: &armnetwork.SubnetPropertiesFormat{
				AddressPrefix:        to.Ptr("10.0.0.0/16"),
				NetworkSecurityGroup: s.Dependencies.SecurityGroup.Resource,
			},
		},
		nil,
	)
	if err != nil {
		return err
	}

	if _, err := poller.PollUntilDone(ctx, nil); err != nil {
		return err
	}

	return s.Read(ctx)
}

func (s *Subnet) Read(ctx context.Context) error {
	response, err := s.client.Services.Subnets.Get(ctx, s.Dependencies.ResourceGroup.Identifier, s.Dependencies.VirtualNetwork.Identifier, s.Identifier, nil)
	if err != nil {
		var e *azcore.ResponseError
		if errors.As(err, &e) && e.RawResponse.StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	s.Resource = &response.Subnet
	return nil
}

func (s *Subnet) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (s *Subnet) Delete(ctx context.Context) error {
	poller, err := s.client.Services.Subnets.BeginDelete(ctx, s.Dependencies.ResourceGroup.Identifier, s.Dependencies.VirtualNetwork.Identifier, s.Identifier, nil)
	if err != nil {
		var e *azcore.ResponseError
		if errors.As(err, &e) && e.RawResponse.StatusCode == 404 {
			return nil
		}
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	s.Resource = nil
	return err
}
