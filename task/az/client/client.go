package client

import (
	"context"
	"errors"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-30/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-06-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-04-01/storage"

	"github.com/Azure/go-autorest/autorest/azure/auth"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/ssh"
)

func New(ctx context.Context, cloud common.Cloud, tags map[string]string) (*Client, error) {
	settings, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		return nil, err
	}

	subscription := settings.GetSubscriptionID()
	if subscription == "" {
		return nil, errors.New("subscription environment variable not found")
	}

	authorizer, err := settings.GetAuthorizer()
	if err != nil {
		return nil, err
	}

	agent := "tpi"

	c := new(Client)
	c.Cloud = cloud
	c.Settings = settings

	for key, value := range tags {
		c.Tags[key] = &value
	}

	region := string(cloud.Region)
	regions := map[string]string{
		"us-east":  "eastus",
		"us-west":  "westus2",
		"eu-north": "northeurope",
		"eu-west":  "westeurope",
	}

	if val, ok := regions[region]; ok {
		region = val
	}

	c.Region = region

	c.Services.Groups = resources.NewGroupsClient(subscription)
	c.Services.Groups.Authorizer = authorizer
	if err := c.Services.Groups.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.SecurityGroups = network.NewSecurityGroupsClient(subscription)
	c.Services.SecurityGroups.Authorizer = authorizer
	if err := c.Services.SecurityGroups.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.PublicIPPrefixes = network.NewPublicIPPrefixesClient(subscription)
	c.Services.PublicIPPrefixes.Authorizer = authorizer
	if err := c.Services.PublicIPPrefixes.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.PublicIPAddresses = network.NewPublicIPAddressesClient(subscription)
	c.Services.PublicIPAddresses.Authorizer = authorizer
	if err := c.Services.PublicIPAddresses.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.VirtualNetworks = network.NewVirtualNetworksClient(subscription)
	c.Services.VirtualNetworks.Authorizer = authorizer
	if err := c.Services.VirtualNetworks.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.Subnets = network.NewSubnetsClient(subscription)
	c.Services.Subnets.Authorizer = authorizer
	if err := c.Services.Subnets.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.Interfaces = network.NewInterfacesClient(subscription)
	c.Services.Interfaces.Authorizer = authorizer
	if err := c.Services.Interfaces.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.VirtualMachines = compute.NewVirtualMachinesClient(subscription)
	c.Services.VirtualMachines.Authorizer = authorizer
	if err := c.Services.VirtualMachines.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.VirtualMachineScaleSets = compute.NewVirtualMachineScaleSetsClient(subscription)
	c.Services.VirtualMachineScaleSets.Authorizer = authorizer
	if err := c.Services.VirtualMachineScaleSets.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.VirtualMachineScaleSetVMs = compute.NewVirtualMachineScaleSetVMsClient(subscription)
	c.Services.VirtualMachineScaleSetVMs.Authorizer = authorizer
	if err := c.Services.VirtualMachineScaleSetVMs.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.StorageAccounts = storage.NewAccountsClient(subscription)
	c.Services.StorageAccounts.Authorizer = authorizer
	if err := c.Services.StorageAccounts.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.BlobContainers = storage.NewBlobContainersClient(subscription)
	c.Services.BlobContainers.Authorizer = authorizer
	if err := c.Services.BlobContainers.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	return c, nil
}

type Client struct {
	Cloud    common.Cloud
	Region   string
	Tags     map[string]*string
	Settings auth.EnvironmentSettings
	Services struct {
		Groups                    resources.GroupsClient
		SecurityGroups            network.SecurityGroupsClient
		PublicIPPrefixes          network.PublicIPPrefixesClient
		PublicIPAddresses         network.PublicIPAddressesClient
		VirtualNetworks           network.VirtualNetworksClient
		Subnets                   network.SubnetsClient
		Interfaces                network.InterfacesClient
		VirtualMachines           compute.VirtualMachinesClient
		VirtualMachineScaleSets   compute.VirtualMachineScaleSetsClient
		VirtualMachineScaleSetVMs compute.VirtualMachineScaleSetVMsClient
		StorageAccounts           storage.AccountsClient
		BlobContainers            storage.BlobContainersClient
	}
}

func (c *Client) GetKeyPair(ctx context.Context) (*ssh.DeterministicSSHKeyPair, error) {
	credentials, err := c.Settings.GetClientCredentials()
	if err != nil {
		return nil, err
	}

	if len(credentials.ClientSecret) == 0 {
		return nil, errors.New("unable to find client secret")
	}

	return ssh.NewDeterministicSSHKeyPair(credentials.ClientSecret, credentials.ClientID)
}
