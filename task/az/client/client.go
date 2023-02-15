package client

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-30/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-06-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-04-01/storage"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/ssh"
)

func New(ctx context.Context, cloud common.Cloud, tags map[string]string) (*Client, error) {
	var authorizer autorest.Authorizer

	if azCredentials := cloud.Credentials.AZCredentials; azCredentials != nil {
		au, err := auth.NewClientCredentialsConfig(
			azCredentials.ClientID,
			azCredentials.ClientSecret,
			azCredentials.TenantID,
		).Authorizer()
		if err != nil {
			return nil, err
		}
		authorizer = au
	} else {
		settings, err := auth.GetSettingsFromEnvironment()
		if err != nil {
			return nil, err
		}
		credentials, err := settings.GetClientCredentials()
		if err != nil {
			return nil, err
		}
		authorizer, err = settings.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		cloud.Credentials.AZCredentials = &common.AZCredentials{
			SubscriptionID: settings.GetSubscriptionID(),
			ClientID:       credentials.ClientID,
			ClientSecret:   credentials.ClientSecret,
			TenantID:       credentials.TenantID,
		}
	}

	agent := "tpi"

	c := &Client{
		Cloud: cloud,
	}

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

	if cloud.Credentials.AZCredentials.SubscriptionID == "" {
		return nil, errors.New("subscription environment variable not found")
	}

	c.Region = region

	c.Services.Groups = resources.NewGroupsClient(cloud.Credentials.AZCredentials.SubscriptionID)
	c.Services.Groups.Authorizer = authorizer
	if err := c.Services.Groups.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.SecurityGroups = network.NewSecurityGroupsClient(cloud.Credentials.AZCredentials.SubscriptionID)
	c.Services.SecurityGroups.Authorizer = authorizer
	if err := c.Services.SecurityGroups.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.PublicIPPrefixes = network.NewPublicIPPrefixesClient(cloud.Credentials.AZCredentials.SubscriptionID)
	c.Services.PublicIPPrefixes.Authorizer = authorizer
	if err := c.Services.PublicIPPrefixes.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.PublicIPAddresses = network.NewPublicIPAddressesClient(cloud.Credentials.AZCredentials.SubscriptionID)
	c.Services.PublicIPAddresses.Authorizer = authorizer
	if err := c.Services.PublicIPAddresses.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.VirtualNetworks = network.NewVirtualNetworksClient(cloud.Credentials.AZCredentials.SubscriptionID)
	c.Services.VirtualNetworks.Authorizer = authorizer
	if err := c.Services.VirtualNetworks.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.Subnets = network.NewSubnetsClient(cloud.Credentials.AZCredentials.SubscriptionID)
	c.Services.Subnets.Authorizer = authorizer
	if err := c.Services.Subnets.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.Interfaces = network.NewInterfacesClient(cloud.Credentials.AZCredentials.SubscriptionID)
	c.Services.Interfaces.Authorizer = authorizer
	if err := c.Services.Interfaces.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.VirtualMachines = compute.NewVirtualMachinesClient(cloud.Credentials.AZCredentials.SubscriptionID)
	c.Services.VirtualMachines.Authorizer = authorizer
	if err := c.Services.VirtualMachines.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.VirtualMachineScaleSets = compute.NewVirtualMachineScaleSetsClient(cloud.Credentials.AZCredentials.SubscriptionID)
	c.Services.VirtualMachineScaleSets.Authorizer = authorizer
	if err := c.Services.VirtualMachineScaleSets.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.VirtualMachineScaleSetVMs = compute.NewVirtualMachineScaleSetVMsClient(cloud.Credentials.AZCredentials.SubscriptionID)
	c.Services.VirtualMachineScaleSetVMs.Authorizer = authorizer
	if err := c.Services.VirtualMachineScaleSetVMs.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.StorageAccounts = storage.NewAccountsClient(cloud.Credentials.AZCredentials.SubscriptionID)
	c.Services.StorageAccounts.Authorizer = authorizer
	if err := c.Services.StorageAccounts.AddToUserAgent(agent); err != nil {
		return nil, err
	}

	c.Services.BlobContainers = storage.NewBlobContainersClient(cloud.Credentials.AZCredentials.SubscriptionID)
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
	credentials := c.Cloud.Credentials.AZCredentials

	if len(credentials.ClientSecret) == 0 {
		return nil, errors.New("unable to find client secret")
	}

	return ssh.NewDeterministicSSHKeyPair(credentials.ClientSecret, credentials.ClientID)
}
