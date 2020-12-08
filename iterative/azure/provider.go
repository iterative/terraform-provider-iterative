package azure

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-30/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-06-01/resources"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//ResourceMachineCreate creates AWS instance
func ResourceMachineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")

	region := getRegion(d)
	instanceType := getInstanceType(d)
	keyPublic := d.Get("key_public").(string)
	vmName := d.Get("instance_name").(string)
	hddSize := int32(d.Get("instance_hdd_size").(int))

	image := d.Get("image").(string)
	if image == "" {
		image = "Canonical:UbuntuServer:18.04-LTS:latest"
	}
	imageParts := strings.Split(image, ":")

	publisher := imageParts[0]
	offer := imageParts[1]
	sku := imageParts[2]
	version := imageParts[3]

	gpName := vmName
	nsgName := vmName + "-nsg"
	vnetName := vmName + "-vnet"
	ipName := vmName + "-ip"
	subnetName := vmName + "-sn"
	nicName := vmName + "-nic"
	ipConfigName := vmName + "-ipc"

	username := "ubuntu"

	d.Set("instance_id", vmName)

	groupsClient, err := getGroupsClient(subscriptionID)
	_, err = groupsClient.CreateOrUpdate(
		ctx,
		gpName,
		resources.Group{
			Location: to.StringPtr(region),
		})
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error creating resource group: %v", err),
		})

		return diags
	}

	// securityGroup
	nsgClient, _ := getNsgClient(subscriptionID)
	futureNsg, _ := nsgClient.CreateOrUpdate(
		ctx,
		gpName,
		nsgName,
		network.SecurityGroup{
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
		ResourceMachineDelete(ctx, d, m)

		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Security group is not ready: %v", err),
		})

		return diags
	}

	//ip
	ipClient, err := getIPClient(subscriptionID)
	futureIP, err := ipClient.CreateOrUpdate(
		ctx,
		gpName,
		ipName,
		network.PublicIPAddress{
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
		ResourceMachineDelete(ctx, d, m)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("cannot create pi: %v", err),
		})

		return diags
	}

	vnetClient, err := getVnetClient(subscriptionID)
	futureVnet, err := vnetClient.CreateOrUpdate(
		ctx,
		gpName,
		vnetName,
		network.VirtualNetwork{
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
		ResourceMachineDelete(ctx, d, m)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("cannot create vnet: %v", err),
		})

		return diags
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
		ResourceMachineDelete(ctx, d, m)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("cannot create subnet: %v", err),
		})

		return diags
	}
	subnet, err := subnetsClient.Get(ctx, gpName, vnetName, subnetName, "")
	if err != nil {
		ResourceMachineDelete(ctx, d, m)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("cannot create subnet: %v", err),
		})

		return diags
	}

	nicClient, _ := getNicClient(subscriptionID)
	nicParams := network.Interface{
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
		ResourceMachineDelete(ctx, d, m)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("cannot create nic: %v", err),
		})

		return diags
	}

	vmClient, _ := getVMClient(subscriptionID)
	futureVM, err := vmClient.CreateOrUpdate(
		ctx,
		gpName,
		vmName,
		compute.VirtualMachine{
			Location: to.StringPtr(region),
			/*
				Plan: &compute.Plan{
					Name:      to.StringPtr(sku),
					Publisher: to.StringPtr(publisher),
					Product:   to.StringPtr(offer),
				},
			*/
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
					ComputerName:  to.StringPtr("iterative"),
					AdminUsername: to.StringPtr(username),
					LinuxConfiguration: &compute.LinuxConfiguration{
						DisablePasswordAuthentication: to.BoolPtr(true),
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
		},
	)
	if err != nil {
		ResourceMachineDelete(ctx, d, m)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("cannot create vm: %v", err),
		})

		return diags
	}
	err = futureVM.WaitForCompletionRef(ctx, vmClient.Client)
	if err != nil {
		ResourceMachineDelete(ctx, d, m)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("cannot create vm: %v", err),
		})

		return diags
	}
	_, err = vmClient.Get(ctx, gpName, vmName, "")
	if err != nil {
		ResourceMachineDelete(ctx, d, m)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("cannot create vm: %v", err),
		})

		return diags
	}

	d.SetId(gpName)
	d.Set("instance_ip", ip.IPAddress)
	d.Set("instance_launch_time", time.Now().String())

	return diags
}

//ResourceMachineDelete deletes Azure instance
func ResourceMachineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	groupsClient, err := getGroupsClient(subscriptionID)
	_, err = groupsClient.Delete(context.Background(), d.Get("instance_id").(string))
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Instance not removed: %v", err),
		})
	}

	return diags
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

func getRegion(d *schema.ResourceData) string {
	instanceRegions := make(map[string]string)
	instanceRegions["us-east"] = "eastus"
	instanceRegions["us-west"] = "westus2"
	instanceRegions["eu-north"] = "northeurope"
	instanceRegions["eu-west"] = "westeurope"

	region := d.Get("region").(string)
	if val, ok := instanceRegions[region]; ok {
		region = val
	}

	return region
}

func getInstanceType(d *schema.ResourceData) string {
	instanceTypes := make(map[string]string)
	instanceTypes["m"] = "Standard_F8s_v2"
	instanceTypes["l"] = "Standard_F32s_v2"
	instanceTypes["xl"] = "Standard_F64s_v2"
	instanceTypes["mk80"] = "Standard_NC6"
	instanceTypes["lk80"] = "Standard_NC12"
	instanceTypes["xlk80"] = "Standard_NC24"
	instanceTypes["mtesla"] = "Standard_NC6s_v3"
	instanceTypes["ltesla"] = "Standard_NC12s_v3"
	instanceTypes["xltesla"] = "Standard_NC24s_v3"

	instanceType := d.Get("instance_type").(string)
	instanceGPU := d.Get("instance_gpu").(string)
	if val, ok := instanceTypes[instanceType+instanceGPU]; ok {
		instanceType = val
	}

	return instanceType
}
