package resources

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"regexp"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

	"github.com/sirupsen/logrus"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
)

func NewVirtualMachineScaleSet(client *client.Client, identifier common.Identifier, resourceGroup *ResourceGroup, subnet *Subnet, securityGroup *SecurityGroup, permissionSet *PermissionSet, credentials *Credentials, task *common.Task) *VirtualMachineScaleSet {
	v := &VirtualMachineScaleSet{
		client:     client,
		Identifier: identifier.Long(),
	}
	v.Attributes.Size = task.Size
	v.Attributes.Environment = task.Environment
	v.Attributes.Firewall = task.Firewall
	v.Attributes.Parallelism = &task.Parallelism
	v.Attributes.Spot = float64(task.Spot)
	v.Dependencies.ResourceGroup = resourceGroup
	v.Dependencies.Subnet = subnet
	v.Dependencies.SecurityGroup = securityGroup
	v.Dependencies.Credentials = credentials
	v.Dependencies.PermissionSet = permissionSet
	return v
}

type VirtualMachineScaleSet struct {
	client     *client.Client
	Identifier string
	Attributes struct {
		Size        common.Size
		Environment common.Environment
		Firewall    common.Firewall
		Parallelism *uint16
		Spot        float64
		Addresses   []net.IP
		Status      common.Status
		Events      []common.Event
	}
	Dependencies struct {
		ResourceGroup *ResourceGroup
		Subnet        *Subnet
		SecurityGroup *SecurityGroup
		Credentials   *Credentials
		PermissionSet *PermissionSet
	}
	Resource *armcompute.VirtualMachineScaleSet
}

func (v *VirtualMachineScaleSet) Create(ctx context.Context) error {
	keyPair, err := v.client.GetKeyPair(ctx)
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

	timeout := time.Now().Add(v.Attributes.Environment.Timeout)
	script, err := machine.Script(v.Attributes.Environment.Script, v.Dependencies.Credentials.Resource, v.Attributes.Environment.Variables, &timeout)
	if err != nil {
		return fmt.Errorf("failed to render machine script: %w", err)
	}

	image := v.Attributes.Environment.Image
	images := map[string]string{
		"ubuntu": "ubuntu@Canonical:0001-com-ubuntu-server-focal:20_04-lts:latest",
		"nvidia": "ubuntu@microsoft-dsvm:ubuntu-2004:2004-gen2:latest",
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
		"s":       "Standard_B1s",
		"m":       "Standard_F8s_v2",
		"l":       "Standard_F32s_v2",
		"xl":      "Standard_F64s_v2",
		"m+t4":    "Standard_NC4as_T4_v3",
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

	settings := armcompute.VirtualMachineScaleSet{
		Tags:     v.client.Tags,
		Location: to.Ptr(v.client.Region),
		SKU: &armcompute.SKU{
			Name:     to.Ptr(size),
			Tier:     to.Ptr("Standard"),
			Capacity: to.Ptr(int64(0)),
		},
		Identity: v.Dependencies.PermissionSet.Resource,
		Properties: &armcompute.VirtualMachineScaleSetProperties{
			UpgradePolicy: &armcompute.UpgradePolicy{
				Mode: to.Ptr(armcompute.UpgradeModeManual),
			},
			VirtualMachineProfile: &armcompute.VirtualMachineScaleSetVMProfile{
				StorageProfile: &armcompute.VirtualMachineScaleSetStorageProfile{
					ImageReference: &armcompute.ImageReference{
						Publisher: to.Ptr(publisher),
						Offer:     to.Ptr(offer),
						SKU:       to.Ptr(sku),
						Version:   to.Ptr(version),
					},
					OSDisk: &armcompute.VirtualMachineScaleSetOSDisk{
						Caching:      to.Ptr(armcompute.CachingTypesReadWrite),
						CreateOption: to.Ptr(armcompute.DiskCreateOptionTypesFromImage),
						ManagedDisk: &armcompute.VirtualMachineScaleSetManagedDiskParameters{
							StorageAccountType: to.Ptr(armcompute.StorageAccountTypesStandardLRS),
						},
					},
				},
				OSProfile: &armcompute.VirtualMachineScaleSetOSProfile{
					ComputerNamePrefix: to.Ptr("tpi"),
					CustomData:         to.Ptr(base64.StdEncoding.EncodeToString([]byte(script))),
					AdminUsername:      to.Ptr(sshUser),
					LinuxConfiguration: &armcompute.LinuxConfiguration{
						SSH: &armcompute.SSHConfiguration{
							PublicKeys: []*armcompute.SSHPublicKey{
								{
									Path:    to.Ptr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", sshUser)),
									KeyData: to.Ptr(publicKey),
								},
							},
						},
					},
				},
				NetworkProfile: &armcompute.VirtualMachineScaleSetNetworkProfile{
					NetworkInterfaceConfigurations: []*armcompute.VirtualMachineScaleSetNetworkConfiguration{
						{
							Name: to.Ptr(v.Identifier),
							Properties: &armcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
								Primary:              to.Ptr(true),
								NetworkSecurityGroup: &armcompute.SubResource{ID: v.Dependencies.SecurityGroup.Resource.ID},
								IPConfigurations: []*armcompute.VirtualMachineScaleSetIPConfiguration{
									{
										Name: to.Ptr(v.Identifier),
										Properties: &armcompute.VirtualMachineScaleSetIPConfigurationProperties{
											Subnet: &armcompute.APIEntityReference{ID: v.Dependencies.Subnet.Resource.ID},
											PublicIPAddressConfiguration: &armcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
												Name: to.Ptr(v.Identifier),
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

	if size := v.Attributes.Size.Storage; size > 0 {
		settings.Properties.VirtualMachineProfile.StorageProfile.OSDisk.DiskSizeGB = to.Ptr(int32(size))
	}

	if plan == "#plan" {
		settings.Plan = &armcompute.Plan{
			Publisher: to.Ptr(publisher),
			Product:   to.Ptr(offer),
			Name:      to.Ptr(sku),
		}
	}

	spot := v.Attributes.Spot
	if spot >= 0 {
		if spot == 0 {
			spot = -1
		}
		*settings.Properties.VirtualMachineProfile.EvictionPolicy = armcompute.VirtualMachineEvictionPolicyTypesDelete
		*settings.Properties.VirtualMachineProfile.Priority = armcompute.VirtualMachinePriorityTypesSpot
		settings.Properties.VirtualMachineProfile.BillingProfile = &armcompute.BillingProfile{
			MaxPrice: to.Ptr(float64(spot)),
		}
	}

	poller, err := v.client.Services.VirtualMachineScaleSets.BeginCreateOrUpdate(
		ctx,
		v.Dependencies.ResourceGroup.Identifier,
		v.Identifier,
		settings,
		nil,
	)
	if err != nil {
		return err
	}

	if _, err := poller.PollUntilDone(ctx, nil); err != nil {
		return err
	}

	return v.Read(ctx)
}

func (v *VirtualMachineScaleSet) Read(ctx context.Context) error {
	response, err := v.client.Services.VirtualMachineScaleSets.Get(ctx, v.Dependencies.ResourceGroup.Identifier, v.Identifier, nil)
	if err != nil {
		var e *azcore.ResponseError
		if errors.As(err, &e) && e.RawResponse.StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	v.Attributes.Events = []common.Event{}
	v.Attributes.Status = common.Status{common.StatusCodeActive: 0}
	scaleSetView, err := v.client.Services.VirtualMachineScaleSets.GetInstanceView(ctx, v.Dependencies.ResourceGroup.Identifier, v.Identifier, nil)
	if err != nil {
		return err
	}
	if scaleSetView.VirtualMachine.StatusesSummary != nil {
		for _, status := range scaleSetView.VirtualMachine.StatusesSummary {
			code := *status.Code
			logrus.Debug("ScaleSet Status Summary:", code, int(*status.Count))
			if code == "ProvisioningState/succeeded" {
				v.Attributes.Status[common.StatusCodeActive] = int(*status.Count)
			}
		}
	}
	if scaleSetView.Statuses != nil {
		for _, status := range scaleSetView.Statuses {
			statusTime := time.Unix(0, 0)
			if status.Time != nil {
				statusTime = *status.Time
			}
			v.Attributes.Events = append(v.Attributes.Events, common.Event{
				Time: statusTime,
				Code: *status.Code,
				Description: []string{
					string(*status.Level),
					*status.DisplayStatus,
					*status.Message,
				},
			})
		}
	}

	v.Attributes.Addresses = []net.IP{}

	for pager := v.client.Services.PublicIPAddresses.NewListPager(v.Dependencies.ResourceGroup.Identifier, nil); pager.More(); {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, machine := range page.PublicIPAddressListResult.Value {
			if address := net.ParseIP(*machine.Properties.IPAddress); address != nil {
				v.Attributes.Addresses = append(v.Attributes.Addresses, address)
			}
		}
	}

	v.Resource = &response.VirtualMachineScaleSet
	return nil
}

func (v *VirtualMachineScaleSet) Update(ctx context.Context) error {
	if err := v.Read(ctx); err != nil {
		return err
	}

	v.Resource.SKU.Capacity = to.Ptr(int64(*v.Attributes.Parallelism))
	poller, err := v.client.Services.VirtualMachineScaleSets.BeginCreateOrUpdate(
		ctx,
		v.Dependencies.ResourceGroup.Identifier,
		v.Identifier,
		*v.Resource,
		nil,
	)
	if err != nil {
		return err
	}

	if _, err := poller.PollUntilDone(ctx, nil); err != nil {
		return err
	}

	return nil
}

func (v *VirtualMachineScaleSet) Delete(ctx context.Context) error {
	poller, err := v.client.Services.VirtualMachineScaleSets.BeginDelete(ctx, v.Dependencies.ResourceGroup.Identifier, v.Identifier, nil)
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
