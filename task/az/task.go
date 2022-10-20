package az

import (
	"context"
	"net"

	"github.com/sirupsen/logrus"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/az/resources"
	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
	"terraform-provider-iterative/task/common/ssh"
)

func List(ctx context.Context, cloud common.Cloud) ([]common.Identifier, error) {
	client, err := client.New(ctx, cloud, nil)
	if err != nil {
		return nil, err
	}

	return resources.ListResourceGroups(ctx, client)
}

func New(ctx context.Context, cloud common.Cloud, identifier common.Identifier, task common.Task) (*Task, error) {
	client, err := client.New(ctx, cloud, cloud.Tags)
	if err != nil {
		return nil, err
	}

	t := new(Task)
	t.Client = client
	t.Identifier = identifier
	t.Attributes = task
	t.DataSources.PermissionSet = resources.NewPermissionSet(
		t.Client,
		t.Attributes.PermissionSet,
	)
	t.Resources.ResourceGroup = resources.NewResourceGroup(
		t.Client,
		t.Identifier,
	)
	t.Resources.StorageAccount = resources.NewStorageAccount(
		t.Client,
		t.Identifier,
		t.Resources.ResourceGroup,
	)
	t.Resources.BlobContainer = resources.NewBlobContainer(
		t.Client,
		t.Identifier,
		t.Resources.ResourceGroup,
		t.Resources.StorageAccount,
	)
	t.DataSources.Credentials = resources.NewCredentials(
		t.Client,
		t.Identifier,
		t.Resources.ResourceGroup,
		t.Resources.StorageAccount,
		t.Resources.BlobContainer,
	)
	t.Resources.VirtualNetwork = resources.NewVirtualNetwork(
		t.Client,
		t.Identifier,
		t.Resources.ResourceGroup,
	)
	t.Resources.SecurityGroup = resources.NewSecurityGroup(
		t.Client,
		t.Identifier,
		t.Resources.ResourceGroup,
		t.Attributes.Firewall,
	)
	t.Resources.Subnet = resources.NewSubnet(
		t.Client,
		t.Identifier,
		t.Resources.ResourceGroup,
		t.Resources.VirtualNetwork,
		t.Resources.SecurityGroup,
	)
	t.Resources.VirtualMachineScaleSet = resources.NewVirtualMachineScaleSet(
		t.Client,
		t.Identifier,
		t.Resources.ResourceGroup,
		t.Resources.Subnet,
		t.Resources.SecurityGroup,
		t.DataSources.PermissionSet,
		t.DataSources.Credentials,
		&t.Attributes,
	)
	return t, nil
}

type Task struct {
	Client      *client.Client
	Identifier  common.Identifier
	Attributes  common.Task
	DataSources struct {
		*resources.Credentials
		*resources.PermissionSet
	}
	Resources struct {
		*resources.ResourceGroup
		*resources.StorageAccount
		*resources.BlobContainer
		*resources.VirtualNetwork
		*resources.Subnet
		*resources.SecurityGroup
		*resources.VirtualMachineScaleSet
	}
}

func (t *Task) Create(ctx context.Context) error {
	logrus.Info("Creating resources...")
	steps := []common.Step{{
		Description: "Creating ResourceGroup...",
		Action:      t.Resources.ResourceGroup.Create,
	}, {
		Description: "Creating StorageAccount...",
		Action:      t.Resources.StorageAccount.Create,
	}, {
		Description: "Creating BlobContainer...",
		Action:      t.Resources.BlobContainer.Create,
	}, {
		Description: "Creating Credentials...",
		Action:      t.DataSources.Credentials.Read,
	}, {
		Description: "Creating VirtualNetwork...",
		Action:      t.Resources.VirtualNetwork.Create,
	}, {
		Description: "Creating SecurityGroup...",
		Action:      t.Resources.SecurityGroup.Create,
	}, {
		Description: "Creating Subnet...",
		Action:      t.Resources.Subnet.Create,
	}, {
		Description: "Creating VirtualMachineScaleSet...",
		Action:      t.Resources.VirtualMachineScaleSet.Create,
	}}
	if t.Attributes.Environment.Directory != "" {
		steps = append(steps, common.Step{
			Description: "Uploading Directory...",
			Action: func(ctx context.Context) error {
				return t.Push(ctx, t.Attributes.Environment.Directory)
			},
		})
	}
	steps = append(steps, common.Step{
		Description: "Starting task...",
		Action:      t.Start,
	})
	if err := common.RunSteps(ctx, steps); err != nil {
		return err
	}
	logrus.Info("Creation completed")
	t.Attributes.Addresses = t.Resources.VirtualMachineScaleSet.Attributes.Addresses
	t.Attributes.Status = t.Resources.VirtualMachineScaleSet.Attributes.Status
	t.Attributes.Events = t.Resources.VirtualMachineScaleSet.Attributes.Events
	return nil
}

func (t *Task) Read(ctx context.Context) error {
	logrus.Info("Reading resources... (this may happen several times)")
	steps := []common.Step{{
		Description: "Reading ResourceGroup...",
		Action:      t.Resources.ResourceGroup.Read,
	}, {
		Description: "Reading StorageAccount...",
		Action:      t.Resources.StorageAccount.Read,
	}, {
		Description: "Reading BlobContainer...",
		Action:      t.Resources.BlobContainer.Read,
	}, {
		Description: "Reading Credentials...",
		Action:      t.DataSources.Credentials.Read,
	}, {
		Description: "Reading VirtualNetwork...",
		Action:      t.Resources.VirtualNetwork.Read,
	}, {
		Description: "Reading SecurityGroup...",
		Action:      t.Resources.SecurityGroup.Read,
	}, {
		Description: "Reading Subnet...",
		Action:      t.Resources.Subnet.Read,
	}, {
		Description: "Reading VirtualMachineScaleSet...",
		Action:      t.Resources.VirtualMachineScaleSet.Read,
	}}
	if err := common.RunSteps(ctx, steps); err != nil {
		return err
	}
	logrus.Info("Read completed")
	t.Attributes.Addresses = t.Resources.VirtualMachineScaleSet.Attributes.Addresses
	t.Attributes.Status = t.Resources.VirtualMachineScaleSet.Attributes.Status
	t.Attributes.Events = t.Resources.VirtualMachineScaleSet.Attributes.Events
	return nil
}

func (t *Task) Delete(ctx context.Context) error {
	logrus.Info("Deleting resources...")
	steps := []common.Step{}

	if t.Read(ctx) == nil {
		if t.Attributes.Environment.DirectoryOut != "" {
			steps = []common.Step{{
				Description: "Downloading Directory...",
				Action: func(ctx context.Context) error {
					err := t.Pull(ctx, t.Attributes.Environment.Directory, t.Attributes.Environment.DirectoryOut)
					if err != nil && err != common.NotFoundError {
						return err
					}
					return nil
				},
			}, {
				Description: "Emptying Bucket...",
				Action: func(ctx context.Context) error {
					err := machine.Delete(ctx, t.DataSources.Credentials.Resource["RCLONE_REMOTE"])
					if err != nil && err != common.NotFoundError {
						return err
					}
					return nil
				},
			}}
		}
	}
	steps = append(steps, []common.Step{{
		Description: "Deleting VirtualMachineScaleSet...",
		Action:      t.Resources.VirtualMachineScaleSet.Delete,
	}, {
		Description: "Deleting Subnet...",
		Action:      t.Resources.Subnet.Delete,
	}, {
		Description: "Deleting SecurityGroup...",
		Action:      t.Resources.SecurityGroup.Delete,
	}, {
		Description: "Deleting VirtualNetwork...",
		Action:      t.Resources.VirtualNetwork.Delete,
	}, {
		Description: "Deleting BlobContainer...",
		Action:      t.Resources.BlobContainer.Delete,
	}, {
		Description: "Deleting StorageAccount...",
		Action:      t.Resources.StorageAccount.Delete,
	}, {
		Description: "Deleting ResourceGroup...",
		Action:      t.Resources.ResourceGroup.Delete,
	}}...)
	if err := common.RunSteps(ctx, steps); err != nil {
		return err
	}
	logrus.Info("Deletion completed")
	return nil
}

func (t *Task) Logs(ctx context.Context) ([]string, error) {
	return machine.Logs(ctx, t.DataSources.Credentials.Resource["RCLONE_REMOTE"])
}

func (t *Task) Pull(ctx context.Context, destination, include string) error {
	if err := t.Read(ctx); err != nil {
		return err
	}

	return machine.Transfer(ctx, t.DataSources.Credentials.Resource["RCLONE_REMOTE"]+"/data", destination, include)
}

func (t *Task) Push(ctx context.Context, source string) error {
	if err := t.Read(ctx); err != nil {
		return err
	}

	return machine.Transfer(ctx, source, t.DataSources.Credentials.Resource["RCLONE_REMOTE"]+"/data", "**")
}

func (t *Task) Start(ctx context.Context) error {
	return t.Resources.VirtualMachineScaleSet.Update(ctx)
}

func (t *Task) Stop(ctx context.Context) error {
	original := t.Attributes.Parallelism
	defer func() { t.Attributes.Parallelism = original }()

	t.Attributes.Parallelism = 0
	return t.Resources.VirtualMachineScaleSet.Update(ctx)
}

func (t *Task) GetAddresses(ctx context.Context) []net.IP {
	return t.Attributes.Addresses
}

func (t *Task) Events(ctx context.Context) []common.Event {
	return t.Attributes.Events
}

func (t *Task) Status(ctx context.Context) (common.Status, error) {
	return machine.Status(ctx, t.DataSources.Credentials.Resource["RCLONE_REMOTE"], t.Attributes.Status)
}

func (t *Task) GetKeyPair(ctx context.Context) (*ssh.DeterministicSSHKeyPair, error) {
	return t.Client.GetKeyPair(ctx)
}

func (t *Task) GetIdentifier(ctx context.Context) common.Identifier {
	return t.Identifier
}
