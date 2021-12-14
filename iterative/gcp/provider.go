package gcp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"terraform-provider-iterative/iterative/utils"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gcp_compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceMachineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) error {
	project, service, err := getProjectService()
	if err != nil {
		return err
	}

	networkName := "iterative"
	instanceName := d.Get("name").(string)
	instanceZone := getRegion(d.Get("region").(string))
	instanceHddSize := int64(d.Get("instance_hdd_size").(int))
	instancePublicSshKey := fmt.Sprintf("%s:%s %s\n", "ubuntu", strings.TrimSpace(d.Get("ssh_public").(string)), "ubuntu")
	instanceServiceAccount := d.Get("instance_permission_set").(string)

	instanceMetadata := map[string]string{}
	for key, value := range d.Get("metadata").(map[string]interface{}) {
		instanceMetadata[key] = value.(string)
	}

	instanceIsPreemptible := d.Get("spot").(bool)
	if d.Get("spot_price").(float64) != -1 {
		return errors.New("Google Cloud preemptible instances don't have a bidding price!")
	}

	instanceRawStartupScript, err := base64.StdEncoding.DecodeString(d.Get("startup_script").(string))
	if err != nil {
		return err
	}
	instanceStartupScript := string(instanceRawStartupScript)

	instanceImageString := d.Get("image").(string)
	if instanceImageString == "" {
		instanceImageString = "ubuntu-os-cloud/ubuntu-2004-lts"
	}
	projectRegex := "(?:(?:[-a-z0-9]{1,63}\\.)*(?:[a-z](?:[-a-z0-9]{0,61}[a-z0-9])?):)?(?:[0-9]{1,19}|(?:[a-z0-9](?:[-a-z0-9]{0,61}[a-z0-9])?))"
	if result, err := regexp.MatchString("^"+projectRegex+"/[-_a-zA-Z0-9]+"+"$", instanceImageString); err != nil || !result {
		return errors.New("Malformed image name! Use project/family to select an image")
	}
	instanceImageComponents := strings.Split(instanceImageString, "/")
	instanceImageProject := instanceImageComponents[0]
	instanceImageFamily := instanceImageComponents[1]

	instanceImage, err := service.Images.GetFromFamily(instanceImageProject, instanceImageFamily).Do()
	if err != nil {
		return err
	}

	instanceType, err := getInstanceType(d.Get("instance_type").(string), d.Get("instance_gpu").(string))
	if err != nil {
		return err
	}

	instanceMachineType, err := service.MachineTypes.Get(project, instanceZone, instanceType["machine"]["type"]).Do()
	if err != nil {
		return err
	}

	instanceDiskType, err := service.DiskTypes.Get(project, instanceZone, "pd-balanced").Do()
	if err != nil {
		return err
	}

	instanceHostMaintenanceBehavior := "MIGRATE"
	if instanceIsPreemptible {
		instanceHostMaintenanceBehavior = "TERMINATE"
	}

	instanceAccelerators := []*gcp_compute.AcceleratorConfig{}
	if instanceType["accelerator"]["count"] != "0" {
		acceleratorType, err := service.AcceleratorTypes.Get(project, instanceZone, instanceType["accelerator"]["type"]).Do()
		if err != nil {
			return err
		}

		acceleratorCount, err := strconv.Atoi(instanceType["accelerator"]["count"])
		if err != nil {
			return err
		}

		instanceHostMaintenanceBehavior = "TERMINATE"
		instanceAccelerators = []*gcp_compute.AcceleratorConfig{
			{
				AcceleratorCount: int64(acceleratorCount),
				AcceleratorType:  acceleratorType.SelfLink,
			},
		}
	}

	network, err := service.Networks.Get(project, networkName).Do()
	if err != nil {
		networkDefinition := &gcp_compute.Network{
			Name:                  networkName,
			AutoCreateSubnetworks: true,
			RoutingConfig: &gcp_compute.NetworkRoutingConfig{
				RoutingMode: "REGIONAL",
			},
		}

		networkInsertOperation, err := service.Networks.Insert(project, networkDefinition).Do()
		if err != nil {
			return err
		}

		networkGetOperationCall := service.GlobalOperations.Get(project, networkInsertOperation.Name)
		_, err = waitForOperation(ctx, d.Timeout(schema.TimeoutCreate), networkGetOperationCall.Do)
		if err != nil {
			return err
		}

		network, err = service.Networks.Get(project, networkName).Do()
		if err != nil {
			return err
		}
	}

	firewallEgressDefinition := &gcp_compute.Firewall{
		Name:       instanceName + "-egress",
		Network:    network.SelfLink,
		Direction:  "EGRESS",
		Priority:   1,
		TargetTags: []string{instanceName},
		Allowed: []*gcp_compute.FirewallAllowed{
			{
				IPProtocol: "all",
			},
		},
		DestinationRanges: []string{
			"0.0.0.0/0",
		},
	}

	firewallEgressInsertOperation, err := service.Firewalls.Insert(project, firewallEgressDefinition).Do()
	if err != nil {
		return err
	}

	firewallEgressGetOperationCall := service.GlobalOperations.Get(project, firewallEgressInsertOperation.Name)
	_, err = waitForOperation(ctx, d.Timeout(schema.TimeoutCreate), firewallEgressGetOperationCall.Do)
	if err != nil {
		return err
	}

	firewallIngressDefinition := &gcp_compute.Firewall{
		Name:       instanceName + "-ingress",
		Network:    network.SelfLink,
		Direction:  "INGRESS",
		Priority:   1,
		TargetTags: []string{instanceName},
		Allowed: []*gcp_compute.FirewallAllowed{
			{
				IPProtocol: "tcp",
				Ports: []string{
					"22",
				},
			},
		},
		SourceRanges: []string{
			"0.0.0.0/0",
		},
	}

	firewallIngressInsertOperation, err := service.Firewalls.Insert(project, firewallIngressDefinition).Do()
	if err != nil {
		return err
	}

	firewallIngressGetOperationCall := service.GlobalOperations.Get(project, firewallIngressInsertOperation.Name)
	_, err = waitForOperation(ctx, d.Timeout(schema.TimeoutCreate), firewallIngressGetOperationCall.Do)
	if err != nil {
		return err
	}

	instanceDefinition := &gcp_compute.Instance{
		Name:        instanceName,
		MachineType: instanceMachineType.SelfLink,
		ServiceAccounts: []*gcp_compute.ServiceAccount{
			{
				Email: instanceServiceAccount,
			},
		},
		Disks: []*gcp_compute.AttachedDisk{
			{
				Boot:       true,
				AutoDelete: true,
				Type:       "PERSISTENT",
				Mode:       "READ_WRITE",
				InitializeParams: &gcp_compute.AttachedDiskInitializeParams{
					DiskName:    instanceName,
					SourceImage: instanceImage.SelfLink,
					DiskSizeGb:  instanceHddSize,
					DiskType:    instanceDiskType.SelfLink,
				},
			},
		},
		NetworkInterfaces: []*gcp_compute.NetworkInterface{
			{
				Network: network.SelfLink,
				AccessConfigs: []*gcp_compute.AccessConfig{
					{
						NetworkTier: "STANDARD",
					},
				},
			},
		},
		Tags: &gcp_compute.Tags{
			Items: []string{instanceName},
		},
		Scheduling: &gcp_compute.Scheduling{
			OnHostMaintenance: instanceHostMaintenanceBehavior,
			Preemptible:       instanceIsPreemptible,
		},
		Labels: instanceMetadata,
		Metadata: &gcp_compute.Metadata{
			Items: []*gcp_compute.MetadataItems{
				{
					Key:   "ssh-keys",
					Value: &instancePublicSshKey,
				},
				{
					Key:   "startup-script",
					Value: &instanceStartupScript,
				},
			},
		},
		GuestAccelerators: instanceAccelerators,
	}

	instanceInsertOperation, err := service.Instances.Insert(project, instanceZone, instanceDefinition).Do()
	if err != nil {
		return err
	}

	instanceGetOperationCall := service.ZoneOperations.Get(project, instanceZone, instanceInsertOperation.Name)
	_, err = waitForOperation(ctx, d.Timeout(schema.TimeoutCreate), instanceGetOperationCall.Do)
	if err != nil {
		return err
	}

	instance, err := service.Instances.Get(project, instanceZone, instanceName).Do()
	if err != nil {
		return err
	}

	d.Set("instance_ip", instance.NetworkInterfaces[0].AccessConfigs[0].NatIP)
	d.Set("instance_launch_time", time.Now().String())

	return nil
}

func ResourceMachineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) error {
	project, service, err := getProjectService()
	if err != nil {
		return err
	}

	instanceZone := getRegion(d.Get("region").(string))
	instanceName := d.Get("name").(string)

	service.Instances.Delete(project, instanceZone, instanceName).Do()
	service.Firewalls.Delete(project, instanceName+"-ingress").Do()
	service.Firewalls.Delete(project, instanceName+"-egress").Do()

	return nil
}

func getProjectService() (string, *gcp_compute.Service, error) {
	var credentials *google.Credentials
	var err error

	if credentialsData := []byte(utils.LoadGCPCredentials()); len(credentialsData) > 0 {
		credentials, err = google.CredentialsFromJSON(oauth2.NoContext, credentialsData, gcp_compute.ComputeScope)
	} else {
		credentials, err = google.FindDefaultCredentials(oauth2.NoContext, gcp_compute.ComputeScope)
	}

	if err != nil {
		return "", nil, err
	}

	service, err := gcp_compute.New(oauth2.NewClient(oauth2.NoContext, credentials.TokenSource))
	if err != nil {
		return "", nil, err
	}

	if credentials.ProjectID == "" {
		return "", nil, errors.New("Couldn't extract the project identifier from the given credentials!")
	}

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS_DATA", string(credentials.JSON))
	return credentials.ProjectID, service, nil
}

func waitForOperation(ctx context.Context, timeout time.Duration, function func(...googleapi.CallOption) (*gcp_compute.Operation, error), arguments ...googleapi.CallOption) (*gcp_compute.Operation, error) {
	var result *gcp_compute.Operation

	err := resource.RetryContext(ctx, timeout, func() *resource.RetryError {
		operation, err := function(arguments...)

		log.Printf("[DEBUG] Waiting for operation: (%#v, %#v)", operation, err)

		if err != nil {
			return resource.NonRetryableError(err)
		}

		if operation.Status != "DONE" {
			err := errors.New("Waiting for operation to complete...")
			return resource.RetryableError(err)
		}

		if operation.Error != nil {
			err := fmt.Errorf("Operation error: %#v", *operation.Error.Errors[0])
			return resource.NonRetryableError(err)
		}

		result = operation
		return nil
	})

	return result, err
}

func getRegion(region string) string {
	instanceRegions := make(map[string]string)
	instanceRegions["us-east"] = "us-east1-c"
	instanceRegions["us-west"] = "us-west1-b"
	instanceRegions["eu-north"] = "europe-north1-a"
	instanceRegions["eu-west"] = "europe-west1-d"

	if val, ok := instanceRegions[region]; ok {
		return val
	}

	return region
}

func getInstanceType(instanceType string, instanceGPU string) (map[string]map[string]string, error) {
	instanceTypes := make(map[string]map[string]map[string]string)
	instanceTypes["m"] = map[string]map[string]string{
		"accelerator": {
			"count": "0",
			"type":  "",
		},
		"machine": {
			"type": "e2-custom-8-32768",
		},
	}
	instanceTypes["l"] = map[string]map[string]string{
		"accelerator": {
			"count": "0",
			"type":  "",
		},
		"machine": {
			"type": "e2-custom-32-131072",
		},
	}
	instanceTypes["xl"] = map[string]map[string]string{
		"accelerator": {
			"count": "0",
			"type":  "",
		},
		"machine": {
			"type": "n2-custom-64-262144",
		},
	}
	instanceTypes["m+k80"] = map[string]map[string]string{
		"accelerator": {
			"count": "1",
			"type":  "nvidia-tesla-k80",
		},
		"machine": {
			"type": "custom-8-53248",
		},
	}
	instanceTypes["l+k80"] = map[string]map[string]string{
		"accelerator": {
			"count": "4",
			"type":  "nvidia-tesla-k80",
		},
		"machine": {
			"type": "custom-32-131072",
		},
	}
	instanceTypes["xl+k80"] = map[string]map[string]string{
		"accelerator": {
			"count": "8",
			"type":  "nvidia-tesla-k80",
		},
		"machine": {
			"type": "custom-64-212992-ext",
		},
	}
	instanceTypes["m+v100"] = map[string]map[string]string{
		"accelerator": {
			"count": "1",
			"type":  "nvidia-tesla-v100",
		},
		"machine": {
			"type": "custom-8-65536-ext",
		},
	}
	instanceTypes["l+v100"] = map[string]map[string]string{
		"accelerator": {
			"count": "4",
			"type":  "nvidia-tesla-v100",
		},
		"machine": {
			"type": "custom-32-262144-ext",
		},
	}
	instanceTypes["xl+v100"] = map[string]map[string]string{
		"accelerator": {
			"count": "8",
			"type":  "nvidia-tesla-v100",
		},
		"machine": {
			"type": "custom-64-524288-ext",
		},
	}

	if val, ok := instanceTypes[instanceType+"+"+instanceGPU]; ok {
		return val, nil
	} else if val, ok := instanceTypes[instanceType]; ok && instanceGPU == "" {
		return val, nil
	} else if val, ok := instanceTypes[instanceType]; ok {
		return map[string]map[string]string{
			"accelerator": {
				"count": val["accelerator"]["count"],
				"type":  instanceGPU,
			},
			"machine": {
				"type": val["machine"]["type"],
			},
		}, nil
	}

	if instanceGPU != "" {
		switch instanceGPU {
		case "k80":
			instanceGPU = "nvidia-tesla-k80"
		case "v100":
			instanceGPU = "nvidia-tesla-v100"
		}

		return map[string]map[string]string{
			"accelerator": {
				"count": "1",
				"type":  instanceGPU,
			},
			"machine": {
				"type": instanceType,
			},
		}, nil
	}

	return map[string]map[string]string{
		"accelerator": {
			"count": "0",
			"type":  "",
		},
		"machine": {
			"type": instanceType,
		},
	}, nil
}
