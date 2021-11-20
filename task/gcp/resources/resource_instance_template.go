package resources

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
	"terraform-provider-iterative/task/gcp/client"
)

func NewInstanceTemplate(client *client.Client, identifier common.Identifier, defaultNetwork *DefaultNetwork, firewallRules []*FirewallRule, image *Image, credentials *Credentials, task common.Task) *InstanceTemplate {
	i := new(InstanceTemplate)
	i.Client = client
	i.Identifier = identifier.Long()
	i.Attributes = task
	i.Dependencies.Credentials = credentials
	i.Dependencies.DefaultNetwork = defaultNetwork
	i.Dependencies.FirewallRules = firewallRules
	i.Dependencies.Image = image
	return i
}

type InstanceTemplate struct {
	Client       *client.Client
	Identifier   string
	Attributes   common.Task
	Dependencies struct {
		*DefaultNetwork
		FirewallRules []*FirewallRule
		*Image
		*Credentials
	}
	Resource *compute.InstanceTemplate
}

func (i *InstanceTemplate) Create(ctx context.Context) error {
	keyPair, err := i.Client.GetKeyPair(ctx)
	if err != nil {
		return err
	}

	publicKey, err := keyPair.PublicString()
	if err != nil {
		return err
	}

	formattedPublicKey := fmt.Sprintf("%s:%s host\n", i.Dependencies.Image.Attributes.SSHUser, strings.TrimSpace(publicKey))

	if i.Attributes.Environment.Variables == nil {
		i.Attributes.Environment.Variables = make(map[string]*string)
	}
	for name, value := range *i.Dependencies.Credentials.Resource {
		valueCopy := value
		i.Attributes.Environment.Variables[name] = &valueCopy
	}

	script := machine.Script(i.Attributes.Environment.Script, i.Attributes.Environment.Variables, i.Attributes.Environment.Timeout)

	size := i.Attributes.Size.Machine
	sizes := map[string]string{
		"m":       "e2-custom-8-32768",
		"l":       "e2-custom-32-131072",
		"xl":      "n2-custom-64-262144",
		"m+k80":   "custom-8-53248+nvidia-tesla-k80*1",
		"l+k80":   "custom-32-131072+nvidia-tesla-k80*4",
		"xl+k80":  "custom-64-212992-ext+nvidia-tesla-k80*8",
		"m+v100":  "custom-8-65536-ext+nvidia-tesla-v100*1",
		"l+v100":  "custom-32-262144-ext+nvidia-tesla-v100*4",
		"xl+v100": "custom-64-524288-ext+nvidia-tesla-v100*8",
	}

	if val, ok := sizes[size]; ok {
		size = val
	}

	accelerators := []*compute.AcceleratorConfig{}

	match := regexp.MustCompile(`^([^+]+)(?:\+([^*]+)\*([1-9]\d*))?$`).FindStringSubmatch(size)
	if match == nil {
		return errors.New("invalid machine type")
	}

	machineType := match[1]
	if match[2] != "" {
		acceleratorType := match[2]
		acceleratorCount, err := strconv.Atoi(match[3])
		if err != nil {
			return err
		}
		accelerators = append(accelerators, &compute.AcceleratorConfig{
			AcceleratorCount: int64(acceleratorCount),
			AcceleratorType:  acceleratorType,
		})
	}

	if i.Attributes.Spot > 0 {
		return errors.New("preemptible instances don't have bidding price")
	}
	isPreemptible := i.Attributes.Spot == 0

	hostMaintenanceBehavior := "MIGRATE"
	if isPreemptible || len(accelerators) > 0 {
		hostMaintenanceBehavior = "TERMINATE"
	}

	var firewallRules []string
	for _, rule := range i.Dependencies.FirewallRules {
		firewallRules = append(firewallRules, rule.Identifier)
	}

	definition := &compute.InstanceTemplate{
		Name: i.Identifier,
		Properties: &compute.InstanceProperties{
			MachineType: machineType,
			Disks: []*compute.AttachedDisk{
				{
					Boot:       true,
					AutoDelete: true,
					Type:       "PERSISTENT",
					Mode:       "READ_WRITE",
					InitializeParams: &compute.AttachedDiskInitializeParams{
						SourceImage: i.Dependencies.Image.Resource.SelfLink,
						DiskSizeGb:  int64(i.Attributes.Size.Storage),
						DiskType:    "pd-balanced",
					},
				},
			},
			NetworkInterfaces: []*compute.NetworkInterface{
				{
					Network: i.Dependencies.DefaultNetwork.Resource.SelfLink,
					AccessConfigs: []*compute.AccessConfig{
						{
							Type:        "ONE_TO_ONE_NAT",
							NetworkTier: "STANDARD",
						},
					},
				},
			},
			Tags: &compute.Tags{
				Items: firewallRules,
			},
			Scheduling: &compute.Scheduling{
				OnHostMaintenance: hostMaintenanceBehavior,
				Preemptible:       isPreemptible,
			},
			Labels: i.Client.Tags,
			Metadata: &compute.Metadata{
				Items: []*compute.MetadataItems{
					{
						Key:   "ssh-keys",
						Value: &formattedPublicKey,
					},
					{
						Key:   "startup-script",
						Value: &script,
					},
				},
			},
			GuestAccelerators: accelerators,
		},
	}

	insertOperation, err := i.Client.Services.Compute.InstanceTemplates.Insert(i.Client.Credentials.ProjectID, definition).Do()
	if err != nil {
		if strings.HasSuffix(err.Error(), "alreadyExists") {
			return i.Read(ctx)
		}
		return err
	}

	getOperationCall := i.Client.Services.Compute.GlobalOperations.Get(i.Client.Credentials.ProjectID, insertOperation.Name)
	_, err = waitForOperation(ctx, i.Client.Cloud.Timeouts.Create, 2*time.Second, 32*time.Second, getOperationCall.Do)
	if err != nil {
		return err
	}

	return i.Read(ctx)
}

func (i *InstanceTemplate) Read(ctx context.Context) error {
	template, err := i.Client.Services.Compute.InstanceTemplates.Get(i.Client.Credentials.ProjectID, i.Identifier).Do()
	if err != nil {
		var e *googleapi.Error
		if errors.As(err, &e) && e.Code == 404 {
			return common.NotFoundError
		}
		return err
	}

	i.Resource = template
	return nil
}

func (i *InstanceTemplate) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (i *InstanceTemplate) Delete(ctx context.Context) error {
	deleteOperationCall := i.Client.Services.Compute.InstanceTemplates.Delete(i.Client.Credentials.ProjectID, i.Identifier)
	_, err := waitForOperation(ctx, i.Client.Cloud.Timeouts.Delete, 2*time.Second, 32*time.Second, deleteOperationCall.Do)
	if err != nil {
		var e *googleapi.Error
		if !errors.As(err, &e) || e.Code != 404 {
			return err
		}
	}

	i.Resource = nil
	return nil
}
