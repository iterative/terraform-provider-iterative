package gcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	instanceId := d.Id()
	instanceZone := getRegion(d.Get("region").(string))
	instanceHddSize := int64(d.Get("instance_hdd_size").(int))
	instancePublicSshKey := fmt.Sprintf("%s:%s %s\n", "ubuntu", strings.TrimSpace(d.Get("ssh_public").(string)), "ubuntu")

	serviceAccountEmail, serviceAccountScopes := getServiceAccountData(d.Get("instance_permission_set").(string))

	instanceName := d.Get("name").(string)
	if instanceName == "" {
		instanceName = instanceId
	}

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
		Name:       instanceId + "-egress",
		Network:    network.SelfLink,
		Direction:  "EGRESS",
		Priority:   1,
		TargetTags: []string{instanceId},
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
		Name:       instanceId + "-ingress",
		Network:    network.SelfLink,
		Direction:  "INGRESS",
		Priority:   1,
		TargetTags: []string{instanceId},
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
				Email:  serviceAccountEmail,
				Scopes: serviceAccountScopes,
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
			Items: []string{instanceId},
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
	instanceId := d.Id()
	instanceName := d.Get("name").(string)
	if instanceName == "" {
		instanceName = instanceId
	}

	service.Instances.Delete(project, instanceZone, instanceName).Do()
	service.Firewalls.Delete(project, instanceId+"-ingress").Do()
	service.Firewalls.Delete(project, instanceId+"-egress").Do()

	return nil
}

func LoadGCPCredentials() (*google.Credentials, error) {
	if credentialsData := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_DATA"); credentialsData != "" {
		return google.CredentialsFromJSON(oauth2.NoContext, []byte(credentialsData), gcp_compute.ComputeScope)
	}

	return google.FindDefaultCredentials(oauth2.NoContext, gcp_compute.ComputeScope)
}

func getServiceAccountData(saString string) (string, []string) {
	// ["SA email", "scopes=s1", "s2", ...]
	splitStr := strings.Split(saString, ",")
	serviceAccountEmail := splitStr[0]
	if len(splitStr) == 1 {
		// warn user about scopes?
		return serviceAccountEmail, nil
	}
	// ["scopes=s1", "s2"]
	splitStr[1] = strings.Split(splitStr[1], "=")[1]
	// ["s1", "s2", ...]
	serviceAccountScopes := splitStr[1:]
	return serviceAccountEmail, getCanonicalizedServiceScopes(serviceAccountScopes)
}

func getProjectService() (string, *gcp_compute.Service, error) {
	credentials, err := LoadGCPCredentials()
	if err != nil {
		return "", nil, err
	}
	var tokenSource oauth2.TokenSource
	if token, err := reuseToken(); err == nil && token != nil {
		tokenSource = oauth2.ReuseTokenSource(token, credentials.TokenSource)
	} else {
		tokenSource = credentials.TokenSource
	}
	service, err := gcp_compute.New(oauth2.NewClient(oauth2.NoContext, tokenSource))
	if err != nil {
		return "", nil, err
	}

	if credentials.ProjectID == "" {
		// 	Coerce Credentials to handle GCP OIDC auth
		//	Common ProjectID ENVs:
		//		https://github.com/google-github-actions/auth/blob/b05f71482f54380997bcc43a29ef5007de7789b1/src/main.ts#L187-L191
		//		https://github.com/hashicorp/terraform-provider-google/blob/d6734812e2c6a679334dcb46932f4b92729fa98c/google/provider.go#L64-L73
		coercedProjectID := utils.MultiEnvLoadFirst([]string{
			"CLOUDSDK_CORE_PROJECT",
			"CLOUDSDK_PROJECT",
			"GCLOUD_PROJECT",
			"GCP_PROJECT",
			"GOOGLE_CLOUD_PROJECT",
			"GOOGLE_PROJECT",
		})
		if coercedProjectID == "" {
			// last effort to load
			fromCredentialsID, err := coerceOIDCCredentials(credentials.JSON)
			if err != nil {
				return "", nil, fmt.Errorf("Couldn't extract the project identifier from the given credentials!: [%w]", err)
			}
			coercedProjectID = fromCredentialsID
		}
		credentials.ProjectID = coercedProjectID
	}

	return credentials.ProjectID, service, nil
}
func reuseToken() (*oauth2.Token, error) {
	var token *oauth2.Token
	tokenJSON := os.Getenv("CML_GCP_ACCESS_TOKEN")
	if len(tokenJSON) == 0 {
		return nil, nil
	}
	err := json.Unmarshal([]byte(tokenJSON), &token)
	return token, err
}

func ExtractToken(credentials *google.Credentials) ([]byte, error) {
	token, err := credentials.TokenSource.Token()
	if err != nil {
		return nil, err
	}
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}
	return tokenJSON, nil
}

func coerceOIDCCredentials(credentialsJSON []byte) (string, error) {
	var credentials map[string]interface{}
	if err := json.Unmarshal(credentialsJSON, &credentials); err != nil {
		return "", err
	}

	if url, ok := credentials["service_account_impersonation_url"].(string); ok {
		re := regexp.MustCompile(`^https://iamcredentials\.googleapis\.com/v1/projects/-/serviceAccounts/.+?@(?P<project>.+)\.iam\.gserviceaccount\.com:generateAccessToken$`)
		if match := re.FindStringSubmatch(url); match != nil {
			return match[1], nil
		}
		return "", errors.New("failed to get project identifier from service_account_impersonation_url")
	}

	return "", errors.New("unable to load service_account_impersonation_url")
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
	instanceTypes["s"] = map[string]map[string]string{
		"accelerator": {
			"count": "0",
			"type":  "",
		},
		"machine": {
			"type": "g1-small",
		},
	}
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
	instanceTypes["m+t4"] = map[string]map[string]string{
		"accelerator": {
			"count": "1",
			"type":  "nvidia-tesla-t4",
		},
		"machine": {
			"type": "n1-standard-4",
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

	match := regexp.MustCompile(`^([^+]+)\+([^*]+)\*([1-9]\d*)?$`).FindStringSubmatch(instanceType)
	if match != nil {
		return map[string]map[string]string{
			"accelerator": {
				"count": match[3],
				"type":  match[2],
			},
			"machine": {
				"type": match[1],
			},
		}, nil
	} else if val, ok := instanceTypes[instanceType+"+"+instanceGPU]; ok {
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

// https://github.com/hashicorp/terraform-provider-google/blob/8a362008bd4d36b6a882eb53455f87305e6dff52/google/service_scope.go#L5-L48
func shorthandServiceScopeLookup(scope string) string {
	// This is a convenience map of short names used by the gcloud tool
	// to the GCE auth endpoints they alias to.
	scopeMap := map[string]string{
		"bigquery":              "https://www.googleapis.com/auth/bigquery",
		"cloud-platform":        "https://www.googleapis.com/auth/cloud-platform",
		"cloud-source-repos":    "https://www.googleapis.com/auth/source.full_control",
		"cloud-source-repos-ro": "https://www.googleapis.com/auth/source.read_only",
		"compute-ro":            "https://www.googleapis.com/auth/compute.readonly",
		"compute-rw":            "https://www.googleapis.com/auth/compute",
		"datastore":             "https://www.googleapis.com/auth/datastore",
		"logging-write":         "https://www.googleapis.com/auth/logging.write",
		"monitoring":            "https://www.googleapis.com/auth/monitoring",
		"monitoring-read":       "https://www.googleapis.com/auth/monitoring.read",
		"monitoring-write":      "https://www.googleapis.com/auth/monitoring.write",
		"pubsub":                "https://www.googleapis.com/auth/pubsub",
		"service-control":       "https://www.googleapis.com/auth/servicecontrol",
		"service-management":    "https://www.googleapis.com/auth/service.management.readonly",
		"sql":                   "https://www.googleapis.com/auth/sqlservice",
		"sql-admin":             "https://www.googleapis.com/auth/sqlservice.admin",
		"storage-full":          "https://www.googleapis.com/auth/devstorage.full_control",
		"storage-ro":            "https://www.googleapis.com/auth/devstorage.read_only",
		"storage-rw":            "https://www.googleapis.com/auth/devstorage.read_write",
		"taskqueue":             "https://www.googleapis.com/auth/taskqueue",
		"trace":                 "https://www.googleapis.com/auth/trace.append",
		"useraccounts-ro":       "https://www.googleapis.com/auth/cloud.useraccounts.readonly",
		"useraccounts-rw":       "https://www.googleapis.com/auth/cloud.useraccounts",
		"userinfo-email":        "https://www.googleapis.com/auth/userinfo.email",
	}
	if matchedURL, ok := scopeMap[scope]; ok {
		return matchedURL
	}
	return scope
}
func getCanonicalizedServiceScopes(scopes []string) []string {
	cs := make([]string, len(scopes))
	for i, scope := range scopes {
		cs[i] = shorthandServiceScopeLookup(scope)
	}
	return cs
}
