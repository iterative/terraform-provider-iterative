package azure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-30/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-06-01/resources"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//ResourceMachineCreate creates AWS instance
func ResourceMachineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) error {
	subscriptionID, err := subscriptionID()
	if err != nil {
		return err
	}

	username := "ubuntu"
	customData := d.Get("startup_script").(string)
	region := GetRegion(d.Get("region").(string))
	instanceType := getInstanceType(d.Get("instance_type").(string), d.Get("instance_gpu").(string))
	keyPublic := d.Get("ssh_public").(string)
	hddSize := int32(d.Get("instance_hdd_size").(int))
	spot := d.Get("spot").(bool)
	spotPrice := d.Get("spot_price").(float64)

	metadata := map[string]*string{}
	for key, value := range d.Get("metadata").(map[string]interface{}) {
		stringValue := value.(string)
		metadata[key] = &stringValue
	}

	image := d.Get("image").(string)
	if image == "" {
		image = "Canonical:UbuntuServer:18.04-LTS:latest"
		//image = "iterative:CML:v1.0.0:1.0.0"
	}

	imageParts := strings.Split(image, ":")
	publisher := imageParts[0]
	offer := imageParts[1]
	sku := imageParts[2]
	version := imageParts[3]

	vmName := d.Get("name").(string)
	gpName := d.Id()
	nsgName := gpName + "-nsg"
	vnetName := gpName + "-vnet"
	ipName := gpName + "-ip"
	subnetName := gpName + "-sn"
	nicName := gpName + "-nic"
	ipConfigName := gpName + "-ipc"

	groupsClient, err := getGroupsClient(subscriptionID)
	_, err = groupsClient.CreateOrUpdate(
		ctx,
		gpName,
		resources.Group{
			Location: to.StringPtr(region),
			Tags:     metadata,
		})
	if err != nil {
		return err
	}

	// securityGroup
	nsgClient, _ := getNsgClient(subscriptionID)
	futureNsg, _ := nsgClient.CreateOrUpdate(
		ctx,
		gpName,
		nsgName,
		network.SecurityGroup{
			Tags:     metadata,
			Location: to.StringPtr(region),
			SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
				SecurityRules: &[]network.SecurityRule{
					{
						Name: to.StringPtr("allow_ssh"),
						SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
							Protocol:                 network.SecurityRuleProtocolTCP,
							SourceAddressPrefix:      to.StringPtr("0.0.0.0/0"),
							SourcePortRange:          to.StringPtr("1-65535"),
							DestinationAddressPrefix: to.StringPtr("0.0.0.0/0"),
							DestinationPortRange:     to.StringPtr("1-65535"),
							Access:                   network.SecurityRuleAccessAllow,
							Direction:                network.SecurityRuleDirectionInbound,
							Priority:                 to.Int32Ptr(100),
						},
					},
				},
			},
		},
	)
	futureNsg.WaitForCompletionRef(ctx, nsgClient.Client)
	nsg, err := nsgClient.Get(ctx, gpName, nsgName, "")
	if err != nil {
		return err
	}

	//ip
	ipClient, err := getIPClient(subscriptionID)
	futureIP, err := ipClient.CreateOrUpdate(
		ctx,
		gpName,
		ipName,
		network.PublicIPAddress{
			Tags:     metadata,
			Name:     to.StringPtr(ipName),
			Location: to.StringPtr(region),
			PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
				PublicIPAddressVersion:   network.IPv4,
				PublicIPAllocationMethod: network.Static,
			},
		},
	)
	futureIP.WaitForCompletionRef(ctx, ipClient.Client)
	ip, err := ipClient.Get(ctx, gpName, ipName, "")
	if err != nil {
		return err
	}

	vnetClient, err := getVnetClient(subscriptionID)
	futureVnet, err := vnetClient.CreateOrUpdate(
		ctx,
		gpName,
		vnetName,
		network.VirtualNetwork{
			Tags:     metadata,
			Location: to.StringPtr(region),
			VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
				AddressSpace: &network.AddressSpace{
					AddressPrefixes: &[]string{"10.0.0.0/8"},
				},
			},
		})
	futureVnet.WaitForCompletionRef(ctx, vnetClient.Client)
	_, err = vnetClient.Get(ctx, gpName, vnetName, "")
	if err != nil {
		return err
	}

	// subnet
	subnetsClient, err := getSubnetsClient(subscriptionID)
	futureSubnet, err := subnetsClient.CreateOrUpdate(
		ctx,
		gpName,
		vnetName,
		subnetName,
		network.Subnet{
			SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
				AddressPrefix:        to.StringPtr("10.0.0.0/16"),
				NetworkSecurityGroup: &nsg,
			},
		})
	futureSubnet.WaitForCompletionRef(ctx, subnetsClient.Client)
	if err != nil {
		return err
	}
	subnet, err := subnetsClient.Get(ctx, gpName, vnetName, subnetName, "")
	if err != nil {
		return err
	}

	nicClient, _ := getNicClient(subscriptionID)
	nicParams := network.Interface{
		Tags:     metadata,
		Name:     to.StringPtr(nicName),
		Location: to.StringPtr(region),
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					Name: to.StringPtr(ipConfigName),
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						PrivateIPAllocationMethod: network.Dynamic,
						Subnet:                    &subnet,
						PublicIPAddress:           &ip,
					},
				},
			},
		},
	}
	nicParams.NetworkSecurityGroup = &nsg
	futureNic, err := nicClient.CreateOrUpdate(ctx, gpName, nicName, nicParams)
	futureNic.WaitForCompletionRef(ctx, nicClient.Client)
	nic, err := nicClient.Get(ctx, gpName, nicName, "")
	if err != nil {
		return err
	}

	vmClient, _ := getVMClient(subscriptionID)
	vmSettings := compute.VirtualMachine{
		Tags:     metadata,
		Location: to.StringPtr(region),
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(instanceType),
			},
			StorageProfile: &compute.StorageProfile{
				ImageReference: &compute.ImageReference{
					Publisher: to.StringPtr(publisher),
					Offer:     to.StringPtr(offer),
					Sku:       to.StringPtr(sku),
					Version:   to.StringPtr(version),
				},
				OsDisk: &compute.OSDisk{
					Name:         to.StringPtr(fmt.Sprintf(vmName + "-hdd")),
					Caching:      compute.CachingTypesReadWrite,
					CreateOption: compute.DiskCreateOptionTypesFromImage,
					DiskSizeGB:   to.Int32Ptr(hddSize),
					ManagedDisk: &compute.ManagedDiskParameters{
						StorageAccountType: compute.StorageAccountTypesStandardLRS,
					},
				},
			},
			OsProfile: &compute.OSProfile{
				CustomData:    to.StringPtr(customData),
				ComputerName:  to.StringPtr("iterative"),
				AdminUsername: to.StringPtr(username),
				LinuxConfiguration: &compute.LinuxConfiguration{
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							{
								Path:    to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", username)),
								KeyData: to.StringPtr(keyPublic),
							},
						},
					},
				},
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						ID: nic.ID,
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
							Primary: to.BoolPtr(true),
						},
					},
				},
			},
		},
	}

	if spot {
		vmSettings.EvictionPolicy = compute.Delete
		vmSettings.Priority = compute.Spot
		vmSettings.BillingProfile = &compute.BillingProfile{
			MaxPrice: to.Float64Ptr(spotPrice),
		}
	}

	futureVM, err := vmClient.CreateOrUpdate(
		ctx,
		gpName,
		vmName,
		vmSettings,
	)
	if err != nil {
		return err
	}
	err = futureVM.WaitForCompletionRef(ctx, vmClient.Client)
	if err != nil {
		return err
	}
	_, err = vmClient.Get(ctx, gpName, vmName, "")
	if err != nil {
		return err
	}

	d.Set("instance_ip", ip.IPAddress)
	d.Set("instance_launch_time", time.Now().String())

	return nil
}

//ResourceMachineDelete deletes Azure instance
func ResourceMachineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) error {
	subscriptionID, err := subscriptionID()
	if err != nil {
		return err
	}
	groupsClient, err := getGroupsClient(subscriptionID)
	if err != nil {
		return err
	}
	groupsClient.Delete(context.Background(), d.Id(), "")
	return nil
}

func getGroupsClient(subscriptionID string) (resources.GroupsClient, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()

	client := resources.NewGroupsClient(subscriptionID)
	client.Authorizer = authorizer
	client.AddToUserAgent("iterative-provider")

	return client, err
}

func getNsgClient(subscriptionID string) (network.SecurityGroupsClient, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	client := network.NewSecurityGroupsClient(subscriptionID)
	client.Authorizer = authorizer
	client.AddToUserAgent("iterative-provider")

	return client, err
}

func getIPClient(subscriptionID string) (network.PublicIPAddressesClient, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	client := network.NewPublicIPAddressesClient(subscriptionID)
	client.Authorizer = authorizer
	client.AddToUserAgent("iterative-provider")

	return client, err
}

func getVnetClient(subscriptionID string) (network.VirtualNetworksClient, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	client := network.NewVirtualNetworksClient(subscriptionID)
	client.Authorizer = authorizer
	client.AddToUserAgent("iterative-provider")

	return client, err
}

func getSubnetsClient(subscriptionID string) (network.SubnetsClient, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	client := network.NewSubnetsClient(subscriptionID)
	client.Authorizer = authorizer
	client.AddToUserAgent("iterative-provider")

	return client, err
}

func getNicClient(subscriptionID string) (network.InterfacesClient, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	client := network.NewInterfacesClient(subscriptionID)
	client.Authorizer = authorizer
	client.AddToUserAgent("iterative-provider")

	return client, err
}

func getVMClient(subscriptionID string) (compute.VirtualMachinesClient, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	client := compute.NewVirtualMachinesClient(subscriptionID)
	client.Authorizer = authorizer
	client.AddToUserAgent("iterative-provider")

	return client, err
}

//GetRegion maps region to real cloud regions
func GetRegion(region string) string {
	instanceRegions := make(map[string]string)
	instanceRegions["us-east"] = "eastus"
	instanceRegions["us-west"] = "westus2"
	instanceRegions["eu-north"] = "northeurope"
	instanceRegions["eu-west"] = "westeurope"

	if val, ok := instanceRegions[region]; ok {
		return val
	}

	return region
}

func getInstanceType(instanceType string, instanceGPU string) string {
	instanceTypes := make(map[string]string)
	instanceTypes["m"] = "Standard_F8s_v2"
	instanceTypes["l"] = "Standard_F32s_v2"
	instanceTypes["xl"] = "Standard_F64s_v2"
	instanceTypes["m+k80"] = "Standard_NC6"
	instanceTypes["l+k80"] = "Standard_NC12"
	instanceTypes["xl+k80"] = "Standard_NC24"
	instanceTypes["m+v100"] = "Standard_NC6s_v3"
	instanceTypes["l+v100"] = "Standard_NC12s_v3"
	instanceTypes["xl+v100"] = "Standard_NC24s_v3"

	if val, ok := instanceTypes[instanceType+"+"+instanceGPU]; ok {
		return val
	} else if val, ok := instanceTypes[instanceType]; ok && instanceGPU == "" {
		return val
	}

	return instanceType
}

func subscriptionID() (string, error) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID != "" {
		return subscriptionID, nil
	}

	return "", errors.New("AZURE_SUBSCRIPTION_ID is not present")
}
