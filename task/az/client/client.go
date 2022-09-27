package client

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
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

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	c := &Client{
		Cloud:    cloud,
		Settings: settings,
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

	c.Region = region

	c.Services.ResourceGroups, err = armresources.NewResourceGroupsClient(subscription, credential, nil)
	if err != nil {
		return nil, err
	}

	c.Services.SecurityGroups, err = armnetwork.NewSecurityGroupsClient(subscription, credential, nil)
	if err != nil {
		return nil, err
	}

	c.Services.PublicIPPrefixes, err = armnetwork.NewPublicIPPrefixesClient(subscription, credential, nil)
	if err != nil {
		return nil, err
	}

	c.Services.PublicIPAddresses, err = armnetwork.NewPublicIPAddressesClient(subscription, credential, nil)
	if err != nil {
		return nil, err
	}

	c.Services.VirtualNetworks, err = armnetwork.NewVirtualNetworksClient(subscription, credential, nil)
	if err != nil {
		return nil, err
	}

	c.Services.Subnets, err = armnetwork.NewSubnetsClient(subscription, credential, nil)
	if err != nil {
		return nil, err
	}

	c.Services.Interfaces, err = armnetwork.NewInterfacesClient(subscription, credential, nil)
	if err != nil {
		return nil, err
	}

	c.Services.VirtualMachines, err = armcompute.NewVirtualMachinesClient(subscription, credential, nil)
	if err != nil {
		return nil, err
	}

	c.Services.VirtualMachineScaleSets, err = armcompute.NewVirtualMachineScaleSetsClient(subscription, credential, nil)
	if err != nil {
		return nil, err
	}

	c.Services.VirtualMachineScaleSetVMs, err = armcompute.NewVirtualMachineScaleSetVMsClient(subscription, credential, nil)
	if err != nil {
		return nil, err
	}

	c.Services.VirtualMachineScaleSets, err = armcompute.NewVirtualMachineScaleSetsClient(subscription, credential, nil)
	if err != nil {
		return nil, err
	}

	c.Services.StorageAccounts, err = armstorage.NewAccountsClient(subscription, credential, nil)
	if err != nil {
		return nil, err
	}

	c.Services.BlobContainers, err = armstorage.NewBlobContainersClient(subscription, credential, nil)
	if err != nil {
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
		ResourceGroups            *armresources.ResourceGroupsClient
		SecurityGroups            *armnetwork.SecurityGroupsClient
		PublicIPPrefixes          *armnetwork.PublicIPPrefixesClient
		PublicIPAddresses         *armnetwork.PublicIPAddressesClient
		VirtualNetworks           *armnetwork.VirtualNetworksClient
		Subnets                   *armnetwork.SubnetsClient
		Interfaces                *armnetwork.InterfacesClient
		VirtualMachines           *armcompute.VirtualMachinesClient
		VirtualMachineScaleSets   *armcompute.VirtualMachineScaleSetsClient
		VirtualMachineScaleSetVMs *armcompute.VirtualMachineScaleSetVMsClient
		StorageAccounts           *armstorage.AccountsClient
		BlobContainers            *armstorage.BlobContainersClient
	}
}

// FIXME: this function is broken with the new credential source
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
