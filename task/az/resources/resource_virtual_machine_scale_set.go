package resources

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-30/compute"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
)

func NewVirtualMachineScaleSet(client *client.Client, identifier common.Identifier, resourceGroup *ResourceGroup, subnet *Subnet, securityGroup *SecurityGroup, credentials *Credentials, task *common.Task) *VirtualMachineScaleSet {
	v := new(VirtualMachineScaleSet)
	v.Client = client
	v.Identifier = identifier.Long()
	v.Attributes.Size = task.Size
	v.Attributes.Environment = task.Environment
	v.Attributes.Firewall = task.Firewall
	v.Attributes.Tags = task.Tags
	v.Attributes.Parallelism = &task.Parallelism
	v.Attributes.Spot = float64(task.Spot)
	v.Dependencies.ResourceGroup = resourceGroup
	v.Dependencies.Subnet = subnet
	v.Dependencies.SecurityGroup = securityGroup
	v.Dependencies.Credentials = credentials
	return v
}

type VirtualMachineScaleSet struct {
	Client     *client.Client
	Identifier string
	Attributes struct {
		Size        common.Size
		Environment common.Environment
		Firewall    common.Firewall
		Tags        map[string]string
		Parallelism *uint16
		Spot        float64
		Addresses   []net.IP
		Status      common.Status
		Events      []common.Event
	}
	Dependencies struct {
		*ResourceGroup
		*Subnet
		*SecurityGroup
		*Credentials
	}
	Resource *compute.VirtualMachineScaleSet
}

func (v *VirtualMachineScaleSet) Create(ctx context.Context) error {
	keyPair, err := v.Client.GetKeyPair(ctx)
	if err != nil {
		return err
	}

	publicKey, err := keyPair.PublicString()
	if err != nil {
		return err
	}

	if v.Attributes.Environment.Variables == nil {
		v.Attributes.Environment.Variables = make(map[string]*string)
	}
	for name, value := range *v.Dependencies.Credentials.Resource {
		valueCopy := value
		v.Attributes.Environment.Variables[name] = &valueCopy
	}

	script := machine.Script(v.Attributes.Environment.Script, v.Attributes.Environment.Variables, v.Attributes.Environment.Timeout)

	image := v.Attributes.Environment.Image
	images := map[string]string{
		"ubuntu": "ubuntu@Canonical:0001-com-ubuntu-server-focal:20_04-lts:latest",
		"nvidia": "ubuntu@nvidia:ngc_base_image_version_b:gen2_21-11-0:latest#plan",
	}
	if val, ok := images[image]; ok {
		image = val
	}

	imageParts := regexp.MustCompile(`^([^@]+)@([^:]+):([^:]+):([^:]+):([^:]+)(:?(#plan)?)$`).FindStringSubmatch(image)
	if imageParts == nil {
		return errors.New("invalid machine image format: use publisher:offer:sku:version")
	}

	sshUser := imageParts[1]
	publisher := imageParts[2]
	offer := imageParts[3]
	sku := imageParts[4]
	version := imageParts[5]
	plan := imageParts[6]

	size := v.Attributes.Size.Machine
	sizes := map[string]string{
		"m":       "Standard_F8s_v2",
		"l":       "Standard_F32s_v2",
		"xl":      "Standard_F64s_v2",
		"m+k80":   "Standard_NC6",
		"l+k80":   "Standard_NC12",
		"xl+k80":  "Standard_NC24",
		"m+v100":  "Standard_NC6s_v3",
		"l+v100":  "Standard_NC12s_v3",
		"xl+v100": "Standard_NC24s_v3",
	}

	if val, ok := sizes[size]; ok {
		size = val
	}

	settings := compute.VirtualMachineScaleSet{
		Tags:     v.Client.Tags,
		Location: to.StringPtr(v.Client.Region),
		Sku: &compute.Sku{
			Name:     to.StringPtr(size),
			Tier:     to.StringPtr("Standard"),
			Capacity: to.Int64Ptr(0),
		},
		VirtualMachineScaleSetProperties: &compute.VirtualMachineScaleSetProperties{
			UpgradePolicy: &compute.UpgradePolicy{
				Mode: compute.UpgradeModeManual,
			},
			VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfile{
				StorageProfile: &compute.VirtualMachineScaleSetStorageProfile{
					ImageReference: &compute.ImageReference{
						Publisher: to.StringPtr(publisher),
						Offer:     to.StringPtr(offer),
						Sku:       to.StringPtr(sku),
						Version:   to.StringPtr(version),
					},
					OsDisk: &compute.VirtualMachineScaleSetOSDisk{
						Caching:      compute.CachingTypesReadWrite,
						CreateOption: compute.DiskCreateOptionTypesFromImage,
						DiskSizeGB:   to.Int32Ptr(int32(v.Attributes.Size.Storage)),
						ManagedDisk: &compute.VirtualMachineScaleSetManagedDiskParameters{
							StorageAccountType: compute.StorageAccountTypesStandardLRS,
						},
					},
				},
				OsProfile: &compute.VirtualMachineScaleSetOSProfile{
					ComputerNamePrefix: to.StringPtr("tpi"),
					CustomData:         to.StringPtr(base64.StdEncoding.EncodeToString([]byte(script))),
					AdminUsername:      to.StringPtr(sshUser),
					LinuxConfiguration: &compute.LinuxConfiguration{
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{
								{
									Path:    to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", sshUser)),
									KeyData: to.StringPtr(publicKey),
								},
							},
						},
					},
				},
				NetworkProfile: &compute.VirtualMachineScaleSetNetworkProfile{
					NetworkInterfaceConfigurations: &[]compute.VirtualMachineScaleSetNetworkConfiguration{
						{
							Name: to.StringPtr(v.Identifier),
							VirtualMachineScaleSetNetworkConfigurationProperties: &compute.VirtualMachineScaleSetNetworkConfigurationProperties{
								Primary:              to.BoolPtr(true),
								NetworkSecurityGroup: &compute.SubResource{ID: v.Dependencies.SecurityGroup.Resource.ID},
								IPConfigurations: &[]compute.VirtualMachineScaleSetIPConfiguration{
									{
										Name: to.StringPtr(v.Identifier),
										VirtualMachineScaleSetIPConfigurationProperties: &compute.VirtualMachineScaleSetIPConfigurationProperties{
											Subnet: &compute.APIEntityReference{ID: v.Dependencies.Subnet.Resource.ID},
											PublicIPAddressConfiguration: &compute.VirtualMachineScaleSetPublicIPAddressConfiguration{
												Name: to.StringPtr(v.Identifier),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if plan == "#plan" {
		settings.Plan = &compute.Plan{
			Publisher: to.StringPtr(publisher),
			Product:   to.StringPtr(offer),
			Name:      to.StringPtr(sku),
		}
	}

	spot := v.Attributes.Spot
	if spot >= 0 {
		if spot == 0 {
			spot = -1
		}
		settings.VirtualMachineScaleSetProperties.VirtualMachineProfile.EvictionPolicy = compute.Delete
		settings.VirtualMachineScaleSetProperties.VirtualMachineProfile.Priority = compute.Spot
		settings.VirtualMachineScaleSetProperties.VirtualMachineProfile.BillingProfile = &compute.BillingProfile{
			MaxPrice: to.Float64Ptr(float64(spot)),
		}
	}

	future, err := v.Client.Services.VirtualMachineScaleSets.CreateOrUpdate(
		ctx,
		v.Dependencies.ResourceGroup.Identifier,
		v.Identifier,
		settings,
	)
	if err != nil {
		return err
	}

	if err := future.WaitForCompletionRef(ctx, v.Client.Services.VirtualMachineScaleSets.Client); err != nil {
		return err
	}

	return v.Read(ctx)
}

func (v *VirtualMachineScaleSet) Read(ctx context.Context) error {
	scaleSet, err := v.Client.Services.VirtualMachineScaleSets.Get(ctx, v.Dependencies.ResourceGroup.Identifier, v.Identifier)
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	v.Attributes.Events = []common.Event{}
	v.Attributes.Status = common.Status{common.StatusCodeActive: 0}
	scaleSetView, err := v.Client.Services.VirtualMachineScaleSets.GetInstanceView(ctx, v.Dependencies.ResourceGroup.Identifier, v.Identifier)
	if err != nil {
		return err
	}
	if scaleSetView.VirtualMachine.StatusesSummary != nil {
		for _, status := range *scaleSetView.VirtualMachine.StatusesSummary {
			code := to.String(status.Code)
			if code == "ProvisioningState/succeeded" {
				v.Attributes.Status[common.StatusCodeActive] = int(to.Int32(status.Count))
			}
		}
	}
	if scaleSetView.Statuses != nil {
		for _, status := range *scaleSetView.Statuses {
			v.Attributes.Events = append(v.Attributes.Events, common.Event{
				Time: status.Time.Time,
				Code: to.String(status.Code),
				Description: []string{
					string(status.Level),
					to.String(status.DisplayStatus),
					to.String(status.Message),
				},
			})
		}
	}

	v.Attributes.Addresses = []net.IP{}
	machineListPages, err := v.Client.Services.PublicIPAddresses.ListVirtualMachineScaleSetPublicIPAddresses(ctx, v.Dependencies.ResourceGroup.Identifier, v.Identifier)
	if err != nil {
		return err
	}

	for machineListPages.NotDone() {
		for _, machine := range machineListPages.Values() {
			if address := net.ParseIP(to.String(machine.PublicIPAddressPropertiesFormat.IPAddress)); address != nil {
				v.Attributes.Addresses = append(v.Attributes.Addresses, address)
			}
		}
		if err := machineListPages.NextWithContext(ctx); err != nil {
			return err
		}
	}

	v.Resource = &scaleSet
	return nil
}

func (v *VirtualMachineScaleSet) Update(ctx context.Context) error {
	if err := v.Read(ctx); err != nil {
		return err
	}

	v.Resource.Sku.Capacity = to.Int64Ptr(int64(*v.Attributes.Parallelism))
	future, err := v.Client.Services.VirtualMachineScaleSets.CreateOrUpdate(
		ctx,
		v.Dependencies.ResourceGroup.Identifier,
		v.Identifier,
		*v.Resource,
	)
	if err != nil {
		return err
	}

	if err := future.WaitForCompletionRef(ctx, v.Client.Services.VirtualMachineScaleSets.Client); err != nil {
		return err
	}

	return nil
}

func (v *VirtualMachineScaleSet) Delete(ctx context.Context) error {
	future, err := v.Client.Services.VirtualMachineScaleSets.Delete(ctx, v.Dependencies.ResourceGroup.Identifier, v.Identifier)
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return nil
		}
		return err
	}

	if err := future.WaitForCompletionRef(ctx, v.Client.Services.VirtualMachineScaleSets.Client); err != nil {
		return err
	}

	v.Resource = nil
	return nil
}
