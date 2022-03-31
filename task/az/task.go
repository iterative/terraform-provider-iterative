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

func New(ctx context.Context, cloud common.Cloud, identifier common.Identifier, task common.Task) (*Task, error) {
	client, err := client.New(ctx, cloud, task.Tags)
	if err != nil {
		return nil, err
	}

	t := new(Task)
	t.Client = client
	t.Identifier = identifier
	t.Attributes = task
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
	logrus.Info("Creating ResourceGroup...")
	if err := t.Resources.ResourceGroup.Create(ctx); err != nil {
		return err
	}
	logrus.Info("Creating StorageAccount...")
	if err := t.Resources.StorageAccount.Create(ctx); err != nil {
		return err
	}
	logrus.Info("Creating BlobContainer...")
	if err := t.Resources.BlobContainer.Create(ctx); err != nil {
		return err
	}
	logrus.Info("Creating Credentials...")
	if err := t.DataSources.Credentials.Read(ctx); err != nil {
		return err
	}
	logrus.Info("Creating VirtualNetwork...")
	if err := t.Resources.VirtualNetwork.Create(ctx); err != nil {
		return err
	}
	logrus.Info("Creating SecurityGroup...")
	if err := t.Resources.SecurityGroup.Create(ctx); err != nil {
		return err
	}
	logrus.Info("Creating Subnet...")
	if err := t.Resources.Subnet.Create(ctx); err != nil {
		return err
	}
	logrus.Info("Creating VirtualMachineScaleSet...")
	if err := t.Resources.VirtualMachineScaleSet.Create(ctx); err != nil {
		return err
	}
	logrus.Info("Uploading Directory...")
	if t.Attributes.Environment.Directory != "" {
		if err := t.Push(ctx, t.Attributes.Environment.Directory); err != nil {
			return err
		}
	}
	logrus.Info("Starting task...")
	if err := t.Start(ctx); err != nil {
		return err
	}
	logrus.Info("Done!")
	t.Attributes.Addresses = t.Resources.VirtualMachineScaleSet.Attributes.Addresses
	t.Attributes.Status = t.Resources.VirtualMachineScaleSet.Attributes.Status
	t.Attributes.Events = t.Resources.VirtualMachineScaleSet.Attributes.Events
	return nil
}

func (t *Task) Read(ctx context.Context) error {
	logrus.Info("Reading ResourceGroup...")
	if err := t.Resources.ResourceGroup.Read(ctx); err != nil {
		return err
	}
	logrus.Info("Reading StorageAccount...")
	if err := t.Resources.StorageAccount.Read(ctx); err != nil {
		return err
	}
	logrus.Info("Reading BlobContainer...")
	if err := t.Resources.BlobContainer.Read(ctx); err != nil {
		return err
	}
	logrus.Info("Reading Credentials...")
	if err := t.DataSources.Credentials.Read(ctx); err != nil {
		return err
	}
	logrus.Info("Reading VirtualNetwork...")
	if err := t.Resources.VirtualNetwork.Read(ctx); err != nil {
		return err
	}
	logrus.Info("Reading SecurityGroup...")
	if err := t.Resources.SecurityGroup.Read(ctx); err != nil {
		return err
	}
	logrus.Info("Reading Subnet...")
	if err := t.Resources.Subnet.Read(ctx); err != nil {
		return err
	}
	logrus.Info("Reading VirtualMachineScaleSet...")
	if err := t.Resources.VirtualMachineScaleSet.Read(ctx); err != nil {
		return err
	}
	logrus.Info("Done!")
	t.Attributes.Addresses = t.Resources.VirtualMachineScaleSet.Attributes.Addresses
	t.Attributes.Status = t.Resources.VirtualMachineScaleSet.Attributes.Status
	t.Attributes.Events = t.Resources.VirtualMachineScaleSet.Attributes.Events
	return nil
}

func (t *Task) Delete(ctx context.Context) error {
	logrus.Info("Downloading Directory...")
	if t.Resources.BlobContainer.Read(ctx) == nil {
		if t.Attributes.Environment.DirectoryOut != "" {
			if err := t.Pull(ctx, t.Attributes.Environment.Directory, t.Attributes.Environment.DirectoryOut); err != nil && err != common.NotFoundError {
				return err
			}
		}
		logrus.Info("Emptying Bucket...")
		if err := machine.Delete(ctx, (*t.DataSources.Credentials.Resource)["RCLONE_REMOTE"]); err != nil && err != common.NotFoundError {
			return err
		}
	}
	logrus.Info("Deleting VirtualMachineScaleSet...")
	if err := t.Resources.VirtualMachineScaleSet.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("Deleting Subnet...")
	if err := t.Resources.Subnet.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("Deleting SecurityGroup...")
	if err := t.Resources.SecurityGroup.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("Deleting VirtualNetwork...")
	if err := t.Resources.VirtualNetwork.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("Deleting BlobContainer...")
	if err := t.Resources.BlobContainer.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("Deleting StorageAccount...")
	if err := t.Resources.StorageAccount.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("Deleting ResourceGroup...")
	if err := t.Resources.ResourceGroup.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("Done!")
	return nil
}

func (t *Task) Logs(ctx context.Context) ([]string, error) {
	if err := t.Read(ctx); err != nil {
		return nil, err
	}

	return machine.Logs(ctx, (*t.DataSources.Credentials.Resource)["RCLONE_REMOTE"])
}

func (t *Task) Pull(ctx context.Context, destination, include string) error {
	if err := t.Read(ctx); err != nil {
		return err
	}

	return machine.Transfer(ctx, (*t.DataSources.Credentials.Resource)["RCLONE_REMOTE"]+"/data", destination, include)
}

func (t *Task) Push(ctx context.Context, source string) error {
	if err := t.Read(ctx); err != nil {
		return err
	}

	return machine.Transfer(ctx, source, (*t.DataSources.Credentials.Resource)["RCLONE_REMOTE"]+"/data", "**")
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
	if err := t.Read(ctx); err != nil {
		return nil, err
	}

	return machine.Status(ctx, (*t.DataSources.Credentials.Resource)["RCLONE_REMOTE"], t.Attributes.Status)
}

func (t *Task) GetKeyPair(ctx context.Context) (*ssh.DeterministicSSHKeyPair, error) {
	return t.Client.GetKeyPair(ctx)
}

func (t *Task) GetIdentifier(ctx context.Context) common.Identifier {
	return t.Identifier
}
