package aws

import (
	"context"
	"net"

	"github.com/sirupsen/logrus"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/aws/resources"
	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
	"terraform-provider-iterative/task/common/ssh"
)

func List(ctx context.Context, cloud common.Cloud) ([]common.Identifier, error) {
	client, err := client.New(ctx, cloud, nil)
	if err != nil {
		return nil, err
	}

	return resources.ListBuckets(ctx, client)
}

func New(ctx context.Context, cloud common.Cloud, identifier common.Identifier, task common.Task) (*Task, error) {
	client, err := client.New(ctx, cloud, task.Tags)
	if err != nil {
		return nil, err
	}

	t := new(Task)
	t.Client = client
	t.Identifier = identifier
	t.Attributes = task
	t.DataSources.DefaultVPC = resources.NewDefaultVPC(
		t.Client,
	)
	t.DataSources.DefaultVPCSubnets = resources.NewDefaultVPCSubnets(
		t.Client,
		t.DataSources.DefaultVPC,
	)
	t.DataSources.Image = resources.NewImage(
		t.Client,
		t.Attributes.Environment.Image,
	)
	t.DataSources.PermissionSet = resources.NewPermissionSet(
		t.Client,
		t.Attributes.PermissionSet,
	)
	t.Resources.Bucket = resources.NewBucket(
		t.Client,
		t.Identifier,
	)
	t.DataSources.Credentials = resources.NewCredentials(
		t.Client,
		t.Identifier,
		t.Resources.Bucket,
	)
	t.Resources.SecurityGroup = resources.NewSecurityGroup(
		t.Client,
		t.Identifier,
		t.DataSources.DefaultVPC,
		t.Attributes.Firewall,
	)
	t.Resources.KeyPair = resources.NewKeyPair(
		t.Client,
		t.Identifier,
	)
	t.Resources.LaunchTemplate = resources.NewLaunchTemplate(
		t.Client,
		t.Identifier,
		t.Resources.SecurityGroup,
		t.DataSources.PermissionSet,
		t.DataSources.Image,
		t.Resources.KeyPair,
		t.DataSources.Credentials,
		t.Attributes,
	)
	t.Resources.AutoScalingGroup = resources.NewAutoScalingGroup(
		t.Client,
		t.Identifier,
		t.DataSources.DefaultVPCSubnets,
		t.Resources.LaunchTemplate,
		&t.Attributes.Parallelism,
		t.Attributes.Spot,
	)
	return t, nil
}

type Task struct {
	Client      *client.Client
	Identifier  common.Identifier
	Attributes  common.Task
	DataSources struct {
		*resources.DefaultVPC
		*resources.DefaultVPCSubnets
		*resources.Image
		*resources.Credentials
		*resources.PermissionSet
	}
	Resources struct {
		*resources.Bucket
		*resources.SecurityGroup
		*resources.KeyPair
		*resources.LaunchTemplate
		*resources.AutoScalingGroup
	}
}

func (t *Task) Create(ctx context.Context) error {
	logrus.Info("Creating resources...")
	logrus.Info("[1/12] Parsing PermissionSet...")
	if err := t.DataSources.PermissionSet.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[2/12] Importing DefaultVPC...")
	if err := t.DataSources.DefaultVPC.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[3/12] Importing DefaultVPCSubnets...")
	if err := t.DataSources.DefaultVPCSubnets.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[4/12] Reading Image...")
	if err := t.DataSources.Image.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[5/12] Creating Bucket...")
	if err := t.Resources.Bucket.Create(ctx); err != nil {
		return err
	}
	logrus.Info("[6/12] Creating SecurityGroup...")
	if err := t.Resources.SecurityGroup.Create(ctx); err != nil {
		return err
	}
	logrus.Info("[7/12] Creating KeyPair...")
	if err := t.Resources.KeyPair.Create(ctx); err != nil {
		return err
	}
	logrus.Info("[8/12] Reading Credentials...")
	if err := t.DataSources.Credentials.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[9/12] Creating LaunchTemplate...")
	if err := t.Resources.LaunchTemplate.Create(ctx); err != nil {
		return err
	}
	logrus.Info("[10/12] Creating AutoScalingGroup...")
	if err := t.Resources.AutoScalingGroup.Create(ctx); err != nil {
		return err
	}
	logrus.Info("[11/12] Uploading Directory...")
	if t.Attributes.Environment.Directory != "" {
		if err := t.Push(ctx, t.Attributes.Environment.Directory); err != nil {
			return err
		}
	}
	logrus.Info("[12/12] Starting task...")
	if err := t.Start(ctx); err != nil {
		return err
	}
	logrus.Info("Creation completed")
	t.Attributes.Addresses = t.Resources.AutoScalingGroup.Attributes.Addresses
	t.Attributes.Status = t.Resources.AutoScalingGroup.Attributes.Status
	t.Attributes.Events = t.Resources.AutoScalingGroup.Attributes.Events
	return nil
}

func (t *Task) Read(ctx context.Context) error {
	logrus.Info("Reading resources... (this may happen several times)")
	logrus.Info("[1/9] Reading DefaultVPC...")
	if err := t.DataSources.DefaultVPC.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[2/9] Reading DefaultVPCSubnets...")
	if err := t.DataSources.DefaultVPCSubnets.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[3/9] Reading Image...")
	if err := t.DataSources.Image.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[4/9] Reading Bucket...")
	if err := t.Resources.Bucket.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[5/9] Reading SecurityGroup...")
	if err := t.Resources.SecurityGroup.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[6/9] Reading KeyPair...")
	if err := t.Resources.KeyPair.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[7/9] Reading Credentials...")
	if err := t.DataSources.Credentials.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[8/9] Reading LaunchTemplate...")
	if err := t.Resources.LaunchTemplate.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[9/9] Reading AutoScalingGroup...")
	if err := t.Resources.AutoScalingGroup.Read(ctx); err != nil {
		return err
	}
	logrus.Info("Read completed")
	t.Attributes.Addresses = t.Resources.AutoScalingGroup.Attributes.Addresses
	t.Attributes.Status = t.Resources.AutoScalingGroup.Attributes.Status
	t.Attributes.Events = t.Resources.AutoScalingGroup.Attributes.Events
	return nil
}

func (t *Task) Delete(ctx context.Context) error {
	logrus.Info("Deleting resources...")
	logrus.Info("[1/8] Downloading Directory...")
	if t.Read(ctx) == nil {
		if t.Attributes.Environment.DirectoryOut != "" {
			if err := t.Pull(ctx, t.Attributes.Environment.Directory, t.Attributes.Environment.DirectoryOut); err != nil && err != common.NotFoundError {
				return err
			}
		}
		logrus.Info("[2/8] Emptying Bucket...")

		// TODO: t.DataSources.Credentials.Resource may be nil, check first.
		if err := machine.Delete(ctx, t.DataSources.Credentials.Resource["RCLONE_REMOTE"]); err != nil && err != common.NotFoundError {
			return err
		}
	}
	logrus.Info("[3/8] Deleting AutoScalingGroup...")
	if err := t.Resources.AutoScalingGroup.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("[4/8] Deleting LaunchTemplate...")
	if err := t.Resources.LaunchTemplate.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("[5/8] Deleting KeyPair...")
	if err := t.Resources.KeyPair.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("[6/8] Deleting SecurityGroup...")
	if err := t.Resources.SecurityGroup.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("[7/8] Reading Credentials...")
	if err := t.DataSources.Credentials.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[8/8] Deleting Bucket...")
	if err := t.Resources.Bucket.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("Deletion completed")
	return nil
}

func (t *Task) Logs(ctx context.Context) ([]string, error) {
	if err := t.Read(ctx); err != nil {
		return nil, err
	}

	return machine.Logs(ctx, t.DataSources.Credentials.Resource["RCLONE_REMOTE"])
}

func (t *Task) Pull(ctx context.Context, destination, include string) error {
	if err := t.Read(ctx); err != nil {
		return err
	}

	// TODO: t.DataSources.Credentials.Resource may be nil, check first.
	return machine.Transfer(ctx, t.DataSources.Credentials.Resource["RCLONE_REMOTE"]+"/data", destination, include)
}

func (t *Task) Push(ctx context.Context, source string) error {
	if err := t.Read(ctx); err != nil {
		return err
	}

	// TODO: t.DataSources.Credentials.Resource may be nil, check first.
	return machine.Transfer(ctx, source, t.DataSources.Credentials.Resource["RCLONE_REMOTE"]+"/data", "**")
}

func (t *Task) Start(ctx context.Context) error {
	return t.Resources.AutoScalingGroup.Update(ctx)
}

func (t *Task) Stop(ctx context.Context) error {
	original := t.Attributes.Parallelism
	defer func() { t.Attributes.Parallelism = original }()

	t.Attributes.Parallelism = 0
	return t.Resources.AutoScalingGroup.Update(ctx)
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

	// TODO: t.DataSources.Credentials.Resource may be nil, check first.
	return machine.Status(ctx, t.DataSources.Credentials.Resource["RCLONE_REMOTE"], t.Attributes.Status)
}

func (t *Task) GetKeyPair(ctx context.Context) (*ssh.DeterministicSSHKeyPair, error) {
	return t.Client.GetKeyPair(ctx)
}

func (t *Task) GetIdentifier(ctx context.Context) common.Identifier {
	return t.Identifier
}
