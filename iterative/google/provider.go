package google

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	gcp "google.golang.org/api/compute/v1"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

//ResourceMachineCreate creates GCP instance
func ResourceMachineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) error {
	zone := ""
	network := ""
	subnetwork := ""
	project := ""
	//customData := d.Get("startup_script").(string)

	username := "ubuntu"
	apiURL := "https://www.googleapis.com/compute/v1/projects/"
	zoneURL := apiURL + project + "/zones/" + zone
	globalURL := apiURL + project + "/global"

	instanceName := d.Get("name").(string)
	region := getRegion(d.Get("region").(string))
	instanceType := getInstanceType(d.Get("instance_type").(string), d.Get("instance_gpu").(string))
	keyPublic := d.Get("ssh_public").(string)
	hddSize := int32(d.Get("instance_hdd_size").(int))
	spot := d.Get("spot").(bool)

	image := d.Get("image").(string)
	if image == "" {
		image = "ubuntu-os-cloud/global/images/ubuntu-1804-bionic-v20180705"
	}

	service, err := gcpService()
	if err != nil {
		return err
	}

	instance := &gcp.Instance{
		Description: "",
		Name:        instanceName,
		MachineType: zoneURL + "/machineTypes/" + instanceType,
		Disks: []*gcp.AttachedDisk{
			{
				Boot:       true,
				AutoDelete: true,
				Type:       "PERSISTENT",
				Mode:       "READ_WRITE",
				InitializeParams: &gcp.AttachedDiskInitializeParams{
					DiskName:    instanceName + "-disk",
					SourceImage: apiURL + image,
					DiskSizeGb:  hddSize,
					DiskType:    apiURL + project + "/zones/" + zone + "/diskTypes/pd-standard",
				},
			},
		},
		NetworkInterfaces: []*gcp.NetworkInterface{
			{
				Network:    globalURL + "/networks/" + network,
				Subnetwork: "projects/" + project + "/regions/" + region + "/subnetworks/" + subnetwork,
			},
		},
		ServiceAccounts: []*gcp.ServiceAccount{
			{
				Email:  d.ServiceAccount,
				Scopes: strings.Split(d.Scopes, ","),
			},
		},
		Scheduling: &gcp.Scheduling{
			Preemptible: spot,
		},
	}

	_, err = service.Instances.Insert(project, zone, instance).Do()
	if err != nil {
		return err
	}

	for {
		op, err := service.ZoneOperations.Get(project, zone, instanceName).Do()
		if err != nil {
			return err
		}

		if op.Status == "DONE" {
			if op.Error != nil {
				return fmt.Errorf("Operation error: %v", *op.Error.Errors[0])
			}
			break
		}
		time.Sleep(1 * time.Second)
	}

	instance, err = service.Instances.Get(project, zone, instanceName).Do()
	if err != nil {
		return err
	}

	sshKeys := fmt.Sprintf("%s:%s %s\n", username, strings.TrimSpace(string(keyPublic)), username)
	_, err = service.Instances.SetMetadata(project, zone, instanceName, &gcp.Metadata{
		Fingerprint: instance.Metadata.Fingerprint,
		Items: []*gcp.MetadataItems{
			{
				Key:   "sshKeys",
				Value: &sshKeys,
			},
		},
	}).Do()

	d.SetId(project + "-" + zone + "-" + instanceName)
	d.Set("instance_ip", instance.NetworkInterfaces[0].AccessConfigs[0].NatIP)
	d.Set("instance_launch_time", time.Now().String())

	return err
}

//ResourceMachineDelete deletes Azure instance
func ResourceMachineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) error {
	parts := strings.Split(d.Id(), "-")
	project := parts[0]
	zone := parts[1]
	instanceName := parts[2]

	service, err := gcpService()
	if err != nil {
		return err
	}

	_, err = service.Instances.Delete(project, zone, instanceName).Do()
	if err != nil {
		return err
	}

	for {
		op, err := service.ZoneOperations.Get(project, zone, instanceName).Do()
		if err != nil {
			return err
		}

		if op.Status == "DONE" {
			if op.Error != nil {
				return fmt.Errorf("Operation error: %v", *op.Error.Errors[0])
			}
			break
		}
		time.Sleep(1 * time.Second)
	}

	return err
}

func gcpService() (*gcp.Service, error) {
	client, err := google.DefaultClient(oauth2.NoContext, gcp.ComputeScope)
	if err != nil {
		return nil, err
	}

	service, err := gcp.New(client)
	if err != nil {
		return nil, err
	}

	return service, nil
}

//ImageRegions provider available image regions
var ImageRegions = []string{}

func getRegion(region string) string {
	instanceRegions := make(map[string]string)

	if val, ok := instanceRegions[region]; ok {
		return val
	}

	return region
}

func getInstanceType(instanceType string, instanceGPU string) string {
	instanceTypes := make(map[string]string)

	if val, ok := instanceTypes[instanceType+instanceGPU]; ok {
		return val
	}

	return instanceType
}
