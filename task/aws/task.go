package aws

import (
	"context"
	"errors"
	"net"

	"github.com/sirupsen/logrus"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/aws/resources"
	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
	"terraform-provider-iterative/task/common/ssh"
)

const s3_region = "region"

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
	var bucketCredentials common.StorageCredentials
	if task.RemoteStorage != nil {
		containerPath := task.RemoteStorage.Path
		// If a subdirectory was not specified, the task id will
		// be used.
		if containerPath == "" {
			containerPath = string(t.Identifier)
		}
		// Container config may override the s3 region.
		if region, ok := task.RemoteStorage.Config[s3_region]; !ok || region == "" {
			task.RemoteStorage.Config[s3_region] = t.Client.Region
		}
		bucket := resources.NewExistingS3Bucket(
			t.Client.Services.S3,
			t.Client.Credentials(),
			*task.RemoteStorage)
		t.DataSources.Bucket = bucket
		bucketCredentials = bucket
	} else {
		bucket := resources.NewBucket(
			t.Client,
			t.Identifier,
		)
		t.Resources.Bucket = bucket
		bucketCredentials = bucket
	}
	t.DataSources.Credentials = resources.NewCredentials(
		t.Client,
		t.Identifier,
		bucketCredentials,
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

// Task represents a task running in aws with all its dependent resources.
type Task struct {
	Client      *client.Client
	Identifier  common.Identifier
	Attributes  common.Task
	DataSources struct {
		DefaultVPC        *resources.DefaultVPC
		DefaultVPCSubnets *resources.DefaultVPCSubnets
		Image             *resources.Image
		Credentials       *resources.Credentials
		PermissionSet     *resources.PermissionSet
		Bucket            *resources.ExistingS3Bucket
	}
	Resources struct {
		Bucket           *resources.Bucket
		SecurityGroup    *resources.SecurityGroup
		KeyPair          *resources.KeyPair
		LaunchTemplate   *resources.LaunchTemplate
		AutoScalingGroup *resources.AutoScalingGroup
	}
}

func (t *Task) Create(ctx context.Context) error {
	logrus.Info("Creating resources...")
	steps := []common.Step{{
		Description: "Parsing PermissionSet...",
		Action:      t.DataSources.PermissionSet.Read,
	}, {
		Description: "Importing DefaultVPC...",
		Action:      t.DataSources.DefaultVPC.Read,
	}, {
		Description: "Importing DefaultVPCSubnets...",
		Action:      t.DataSources.DefaultVPCSubnets.Read,
	}, {
		Description: "Reading Image...",
		Action:      t.DataSources.Image.Read,
	}}
	if t.Resources.Bucket != nil {
		steps = append(steps, common.Step{
			Description: "Creating Bucket...",
			Action:      t.Resources.Bucket.Create,
		})
	} else if t.DataSources.Bucket != nil {
		steps = append(steps, common.Step{
			Description: "Verifying bucket...",
			Action:      t.DataSources.Bucket.Read,
		})
	}
	steps = append(steps, []common.Step{{
		Description: "Creating SecurityGroup...",
		Action:      t.Resources.SecurityGroup.Create,
	}, {
		Description: "Creating KeyPair...",
		Action:      t.Resources.KeyPair.Create,
	}, {
		Description: "Reading Credentials...",
		Action:      t.DataSources.Credentials.Read,
	}, {
		Description: "Creating LaunchTemplate...",
		Action:      t.Resources.LaunchTemplate.Create,
	}, {
		Description: "Creating AutoScalingGroup...",
		Action:      t.Resources.AutoScalingGroup.Create,
	}}...)

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
	t.Attributes.Addresses = t.Resources.AutoScalingGroup.Attributes.Addresses
	t.Attributes.Status = t.Resources.AutoScalingGroup.Attributes.Status
	t.Attributes.Events = t.Resources.AutoScalingGroup.Attributes.Events
	return nil
}

func (t *Task) Read(ctx context.Context) error {
	logrus.Info("Reading resources... (this may happen several times)")
	steps := []common.Step{{
		Description: "Reading DefaultVPC...",
		Action:      t.DataSources.DefaultVPC.Read,
	}, {
		Description: "Reading DefaultVPCSubnets...",
		Action:      t.DataSources.DefaultVPCSubnets.Read,
	}, {
		Description: "Reading Image...",
		Action:      t.DataSources.Image.Read,
	}, {
		Description: "Reading Bucket...",
		Action: func(ctx context.Context) error {
			if t.Resources.Bucket != nil {
				return t.Resources.Bucket.Read(ctx)
			} else if t.DataSources.Bucket != nil {
				return t.DataSources.Bucket.Read(ctx)
			}
			return errors.New("storage misconfigured")
		},
	}, {
		Description: "Reading SecurityGroup...",
		Action:      t.Resources.SecurityGroup.Read,
	}, {
		Description: "Reading KeyPair...",
		Action:      t.Resources.KeyPair.Read,
	}, {
		Description: "Reading Credentials...",
		Action:      t.DataSources.Credentials.Read,
	}, {
		Description: "Reading LaunchTemplate...",
		Action:      t.Resources.LaunchTemplate.Read,
	}, {
		Description: "Reading AutoScalingGroup...",
		Action:      t.Resources.AutoScalingGroup.Read,
	}}
	if err := common.RunSteps(ctx, steps); err != nil {
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
				}}}
		}
		if t.Resources.Bucket != nil {
			steps = append(steps, common.Step{
				Description: "Emptying Bucket...",
				Action: func(ctx context.Context) error {
					err := machine.Delete(ctx, t.DataSources.Credentials.Resource["RCLONE_REMOTE"])
					if err != nil && err != common.NotFoundError {
						return err
					}
					return nil
				}})
		}
	}
	steps = append(steps, []common.Step{{
		Description: "Deleting AutoScalingGroup...",
		Action:      t.Resources.AutoScalingGroup.Delete,
	}, {
		Description: "Deleting LaunchTemplate...",
		Action:      t.Resources.LaunchTemplate.Delete,
	}, {
		Description: "Deleting KeyPair...",
		Action:      t.Resources.KeyPair.Delete,
	}, {
		Description: "Deleting SecurityGroup...",
		Action:      t.Resources.SecurityGroup.Delete,
	}, {
		Description: "Reading Credentials...",
		Action:      t.DataSources.Credentials.Read,
	}}...)
	if t.Resources.Bucket != nil {
		steps = append(steps, common.Step{
			Description: "Deleting Bucket...",
			Action:      t.Resources.Bucket.Delete,
		})
	}
	if err := common.RunSteps(ctx, steps); err != nil {
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

	return machine.Transfer(ctx, t.DataSources.Credentials.Resource["RCLONE_REMOTE"]+"/data", destination, include)
}

func (t *Task) Push(ctx context.Context, source string) error {
	if err := t.Read(ctx); err != nil {
		return err
	}

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

	return machine.Status(ctx, t.DataSources.Credentials.Resource["RCLONE_REMOTE"], t.Attributes.Status)
}

func (t *Task) GetKeyPair(ctx context.Context) (*ssh.DeterministicSSHKeyPair, error) {
	return t.Client.GetKeyPair(ctx)
}

func (t *Task) GetIdentifier(ctx context.Context) common.Identifier {
	return t.Identifier
}
