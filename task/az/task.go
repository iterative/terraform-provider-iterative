package az

import (
	"context"
	"net"

	"github.com/0x2b3bfa0/logrusctx"

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
	var bucketCredentials common.StorageCredentials
	if task.RemoteStorage != nil {
		// If a subdirectory was not specified, the task id will
		// be used.
		if task.RemoteStorage.Path == "" {
			task.RemoteStorage.Path = t.Identifier.Short()
		}
		bucket := resources.NewExistingBlobContainer(t.Client, *task.RemoteStorage)
		t.DataSources.BlobContainer = bucket
		bucketCredentials = bucket
	} else {
		t.Resources.StorageAccount = resources.NewStorageAccount(
			t.Client,
			t.Identifier,
			t.Resources.ResourceGroup,
		)
		blobContainer := resources.NewBlobContainer(
			t.Client,
			t.Identifier,
			t.Resources.ResourceGroup,
			t.Resources.StorageAccount,
		)
		t.Resources.BlobContainer = blobContainer
		bucketCredentials = blobContainer
	}
	t.DataSources.Credentials = resources.NewCredentials(
		t.Client,
		t.Identifier,
		t.Resources.ResourceGroup,
		bucketCredentials,
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
		Credentials   *resources.Credentials
		PermissionSet *resources.PermissionSet
		BlobContainer *resources.ExistingBlobContainer
	}
	Resources struct {
		ResourceGroup          *resources.ResourceGroup
		StorageAccount         *resources.StorageAccount
		BlobContainer          *resources.BlobContainer
		VirtualNetwork         *resources.VirtualNetwork
		Subnet                 *resources.Subnet
		SecurityGroup          *resources.SecurityGroup
		VirtualMachineScaleSet *resources.VirtualMachineScaleSet
	}
}

func (t *Task) Create(ctx context.Context) error {
	logrusctx.Info(ctx, "Creating resources...")
	steps := []common.Step{{
		Description: "Creating ResourceGroup...",
		Action:      t.Resources.ResourceGroup.Create,
	}}
	if t.Resources.BlobContainer != nil {
		steps = append(steps, []common.Step{{
			Description: "Creating StorageAccount...",
			Action:      t.Resources.StorageAccount.Create,
		}, {
			Description: "Creating BlobContainer...",
			Action:      t.Resources.BlobContainer.Create,
		}}...)
	} else if t.DataSources.BlobContainer != nil {
		steps = append(steps, common.Step{
			Description: "Reading BlobContainer...",
			Action:      t.DataSources.BlobContainer.Read,
		})
	}

	steps = append(steps, []common.Step{{
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
	}}...)
	if t.Attributes.Environment.Directory != "" {
		steps = append(steps, common.Step{
			Description: "Uploading Directory...",
			Action:      t.Push,
		})
	}
	steps = append(steps, common.Step{
		Description: "Starting task...",
		Action:      t.Start,
	})
	if err := common.RunSteps(ctx, steps); err != nil {
		return err
	}
	logrusctx.Info(ctx, "Creation completed")
	t.Attributes.Addresses = t.Resources.VirtualMachineScaleSet.Attributes.Addresses
	t.Attributes.Status = t.Resources.VirtualMachineScaleSet.Attributes.Status
	t.Attributes.Events = t.Resources.VirtualMachineScaleSet.Attributes.Events
	return nil
}

func (t *Task) Read(ctx context.Context) error {
	logrusctx.Info(ctx, "Reading resources... (this may happen several times)")
	steps := []common.Step{{
		Description: "Reading ResourceGroup...",
		Action:      t.Resources.ResourceGroup.Read,
	}}
	if t.Resources.BlobContainer != nil {
		steps = append(steps, []common.Step{{
			Description: "Reading StorageAccount...",
			Action:      t.Resources.StorageAccount.Read,
		}, {
			Description: "Reading BlobContainer...",
			Action:      t.Resources.BlobContainer.Read,
		}}...)
	} else {
		steps = append(steps, common.Step{
			Description: "Reading BlobContainer...",
			Action:      t.DataSources.BlobContainer.Read,
		})
	}

	steps = append(steps, []common.Step{{
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
	}}...)
	if err := common.RunSteps(ctx, steps); err != nil {
		return err
	}
	logrusctx.Info(ctx, "Read completed")
	t.Attributes.Addresses = t.Resources.VirtualMachineScaleSet.Attributes.Addresses
	t.Attributes.Status = t.Resources.VirtualMachineScaleSet.Attributes.Status
	t.Attributes.Events = t.Resources.VirtualMachineScaleSet.Attributes.Events
	return nil
}

func (t *Task) Delete(ctx context.Context) error {
	logrusctx.Info(ctx, "Deleting resources...")
	steps := []common.Step{}

	if t.Read(ctx) == nil {
		if t.Attributes.Environment.DirectoryOut != "" {
			steps = []common.Step{{
				Description: "Downloading Directory...",
				Action: func(ctx context.Context) error {
					err := t.Pull(ctx)
					if err != nil && err != common.NotFoundError {
						return err
					}
					return nil
				},
			}}
		}
		if t.Resources.BlobContainer != nil {
			steps = append(steps, common.Step{
				Description: "Emptying Bucket...",
				Action: func(ctx context.Context) error {
					err := machine.Delete(ctx, t.DataSources.Credentials.Resource["RCLONE_REMOTE"])
					if err != nil && err != common.NotFoundError {
						return err
					}
					return nil
				},
			})
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
	}}...)

	if t.Resources.BlobContainer != nil {
		steps = append(steps, []common.Step{{
			Description: "Deleting BlobContainer...",
			Action:      t.Resources.BlobContainer.Delete,
		}, {
			Description: "Deleting StorageAccount...",
			Action:      t.Resources.StorageAccount.Delete,
		}}...)
	}
	steps = append(steps, common.Step{
		Description: "Deleting ResourceGroup...",
		Action:      t.Resources.ResourceGroup.Delete,
	})
	if err := common.RunSteps(ctx, steps); err != nil {
		return err
	}
	logrusctx.Info(ctx, "Deletion completed")
	return nil
}

func (t *Task) Logs(ctx context.Context) ([]string, error) {
	return machine.Logs(ctx, t.DataSources.Credentials.Resource["RCLONE_REMOTE"])
}

// Pull downloads the output directory from remote storage.
func (t *Task) Pull(ctx context.Context) error {
	return machine.Transfer(ctx,
		t.DataSources.Credentials.Resource["RCLONE_REMOTE"]+"/data",
		t.Attributes.Environment.Directory,
		machine.LimitTransfer(
			t.Attributes.Environment.DirectoryOut,
			t.Attributes.Environment.ExcludeList))
}

// Push uploads the work directory to remote storage.
func (t *Task) Push(ctx context.Context) error {
	return machine.Transfer(ctx,
		t.Attributes.Environment.Directory,
		t.DataSources.Credentials.Resource["RCLONE_REMOTE"]+"/data",
		t.Attributes.Environment.ExcludeList,
	)
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
