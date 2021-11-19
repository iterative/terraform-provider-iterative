package resources

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"terraform-provider-iterative/task/gcp/client"
	"terraform-provider-iterative/task/universal"
)

func NewInstanceGroupManager(client *client.Client, identifier universal.Identifier, instanceTemplate *InstanceTemplate, parallelism uint16) *InstanceGroupManager {
	i := new(InstanceGroupManager)
	i.Client = client
	i.Identifier = identifier.Long()
	i.Attributes.Parallelism = parallelism
	i.Dependencies.InstanceTemplate = instanceTemplate
	return i
}

type InstanceGroupManager struct {
	Client     *client.Client
	Identifier string
	Attributes struct {
		Parallelism uint16
		Addresses   []net.IP
		Status      map[string]int
		Events      []universal.Event
	}
	Dependencies struct {
		*InstanceTemplate
	}
	Resource *compute.InstanceGroupManager
}

func (i *InstanceGroupManager) Read(ctx context.Context) error {
	manager, err := i.Client.Services.Compute.InstanceGroupManagers.Get(i.Client.Credentials.ProjectID, i.Client.Region, i.Identifier).Do()
	if err != nil {
		var e *googleapi.Error
		if errors.As(err, &e) && e.Code == 404 {
			return universal.NotFoundError
		}
		return err
	}

	i.Attributes.Events = []universal.Event{}
	errors, err := i.Client.Services.Compute.InstanceGroupManagers.ListErrors(i.Client.Credentials.ProjectID, i.Client.Region, i.Identifier).Do()
	if err != nil {
		return err
	}
	for _, error := range errors.Items {
		timeStamp, err := time.Parse(time.RFC3339, error.Timestamp)
		if err != nil {
			timeStamp = time.Time{}
		}
		i.Attributes.Events = append(i.Attributes.Events, universal.Event{
			Time: timeStamp,
			Code: error.Error.Code,
			Description: []string{
				error.Error.Message,
				error.InstanceActionDetails.Action,
			},
		})
	}

	groupInstances, err := i.Client.Services.Compute.InstanceGroups.ListInstances(i.Client.Credentials.ProjectID, i.Client.Region, i.Identifier, &compute.InstanceGroupsListInstancesRequest{}).Do()
	if err != nil {
		return err
	}

	i.Attributes.Addresses = []net.IP{}
	i.Attributes.Status = make(map[string]int)
	for _, groupInstance := range groupInstances.Items {
		i.Attributes.Status[groupInstance.Status]++
		if groupInstance.Status == "RUNNING" {
			instance, err := i.Client.Services.Compute.Instances.Get(i.Client.Credentials.ProjectID, i.Client.Region, filepath.Base(groupInstance.Instance)).Do()
			if err != nil {
				return err
			}
			i.Attributes.Addresses = append(i.Attributes.Addresses, net.ParseIP(instance.NetworkInterfaces[0].AccessConfigs[0].NatIP))
		}
	}

	i.Resource = manager
	return nil
}

func (i *InstanceGroupManager) Create(ctx context.Context) error {
	definition := &compute.InstanceGroupManager{
		Name:             i.Identifier,
		BaseInstanceName: i.Identifier,
		InstanceTemplate: i.Dependencies.InstanceTemplate.Resource.SelfLink,
		TargetSize:       int64(i.Attributes.Parallelism),
		UpdatePolicy: &compute.InstanceGroupManagerUpdatePolicy{
			MaxSurge: &compute.FixedOrPercent{
				Fixed: 0,
			},
			MaxUnavailable: &compute.FixedOrPercent{
				Fixed: int64(i.Attributes.Parallelism),
			},
		},
	}

	insertOperation, err := i.Client.Services.Compute.InstanceGroupManagers.Insert(i.Client.Credentials.ProjectID, i.Client.Region, definition).Do()
	if err != nil {
		if strings.HasSuffix(err.Error(), "alreadyExists") {
			return i.Read(ctx)
		}
		return err
	}

	getOperationCall := i.Client.Services.Compute.ZoneOperations.Get(i.Client.Credentials.ProjectID, i.Client.Region, insertOperation.Name)
	_, err = waitForOperation(ctx, i.Client.Cloud.Timeouts.Create, 2*time.Second, 32*time.Second, getOperationCall.Do)
	if err != nil {
		return err
	}

	return nil
}

func (i *InstanceGroupManager) Update(ctx context.Context) error {
	insertOperation, err := i.Client.Services.Compute.InstanceGroupManagers.Resize(i.Client.Credentials.ProjectID, i.Client.Region, i.Identifier, int64(i.Attributes.Parallelism)).Do()
	if err != nil {
		return err
	}

	getOperationCall := i.Client.Services.Compute.ZoneOperations.Get(i.Client.Credentials.ProjectID, i.Client.Region, insertOperation.Name)
	_, err = waitForOperation(ctx, i.Client.Cloud.Timeouts.Create, 2*time.Second, 32*time.Second, getOperationCall.Do)
	if err != nil {
		return err
	}

	return nil
}

func (i *InstanceGroupManager) Delete(ctx context.Context) error {
	deleteOperationCall := i.Client.Services.Compute.InstanceGroupManagers.Delete(i.Client.Credentials.ProjectID, i.Client.Region, i.Identifier)
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
