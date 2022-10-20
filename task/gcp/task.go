package gcp

import (
	"context"
	"net"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
	"terraform-provider-iterative/task/common/ssh"
	"terraform-provider-iterative/task/gcp/client"
	"terraform-provider-iterative/task/gcp/resources"
)

func List(ctx context.Context, cloud common.Cloud) ([]common.Identifier, error) {
	client, err := client.New(ctx, cloud, nil)
	if err != nil {
		return nil, err
	}

	return resources.ListBuckets(ctx, client)
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
	t.Resources.Bucket = resources.NewBucket(
		t.Client,
		t.Identifier,
	)
	t.DataSources.Credentials = resources.NewCredentials(
		t.Client,
		t.Identifier,
		t.Resources.Bucket,
	)
	t.DataSources.DefaultNetwork = resources.NewDefaultNetwork(
		t.Client,
	)
	t.Resources.FirewallInternalEgress = resources.NewFirewallRule(
		t.Client,
		t.Identifier,
		t.DataSources.DefaultNetwork,
		common.FirewallRule{Nets: &[]net.IPNet{{IP: net.IP{10, 128, 0, 0}, Mask: net.IPMask{255, 128, 0, 0}}}},
		resources.FirewallRuleDirectionEgress,
		resources.FirewallRuleActionAllow,
		1,
	)
	t.Resources.FirewallInternalIngress = resources.NewFirewallRule(
		t.Client,
		t.Identifier,
		t.DataSources.DefaultNetwork,
		common.FirewallRule{Nets: &[]net.IPNet{{IP: net.IP{10, 128, 0, 0}, Mask: net.IPMask{255, 128, 0, 0}}}},
		resources.FirewallRuleDirectionIngress,
		resources.FirewallRuleActionAllow,
		1,
	)
	t.Resources.FirewallExternalEgress = resources.NewFirewallRule(
		t.Client,
		t.Identifier,
		t.DataSources.DefaultNetwork,
		t.Attributes.Firewall.Egress,
		resources.FirewallRuleDirectionEgress,
		resources.FirewallRuleActionAllow,
		2,
	)
	t.Resources.FirewallExternalIngress = resources.NewFirewallRule(
		t.Client,
		t.Identifier,
		t.DataSources.DefaultNetwork,
		t.Attributes.Firewall.Ingress,
		resources.FirewallRuleDirectionIngress,
		resources.FirewallRuleActionAllow,
		2,
	)
	t.Resources.FirewallDenyEgress = resources.NewFirewallRule(
		t.Client,
		t.Identifier,
		t.DataSources.DefaultNetwork,
		common.FirewallRule{},
		resources.FirewallRuleDirectionEgress,
		resources.FirewallRuleActionDeny,
		3,
	)
	t.Resources.FirewallDenyIngress = resources.NewFirewallRule(
		t.Client,
		t.Identifier,
		t.DataSources.DefaultNetwork,
		common.FirewallRule{},
		resources.FirewallRuleDirectionIngress,
		resources.FirewallRuleActionDeny,
		3,
	)
	t.DataSources.Image = resources.NewImage(
		t.Client,
		t.Attributes.Environment.Image,
	)
	t.Resources.InstanceTemplate = resources.NewInstanceTemplate(
		t.Client,
		t.Identifier,
		t.DataSources.DefaultNetwork,
		[]*resources.FirewallRule{
			t.Resources.FirewallInternalEgress,
			t.Resources.FirewallInternalIngress,
			t.Resources.FirewallExternalEgress,
			t.Resources.FirewallExternalIngress,
			t.Resources.FirewallDenyEgress,
			t.Resources.FirewallDenyIngress,
		},
		t.DataSources.PermissionSet,
		t.DataSources.Image,
		t.DataSources.Credentials,
		t.Attributes,
	)
	t.Resources.InstanceGroupManager = resources.NewInstanceGroupManager(
		t.Client,
		t.Identifier,
		t.Resources.InstanceTemplate,
		&t.Attributes.Parallelism,
	)
	return t, nil
}

type Task struct {
	Client      *client.Client
	Identifier  common.Identifier
	Attributes  common.Task
	DataSources struct {
		*resources.DefaultNetwork
		*resources.Credentials
		*resources.Image
		*resources.PermissionSet
	}
	Resources struct {
		*resources.Bucket
		FirewallInternalIngress *resources.FirewallRule
		FirewallInternalEgress  *resources.FirewallRule
		FirewallExternalIngress *resources.FirewallRule
		FirewallExternalEgress  *resources.FirewallRule
		FirewallDenyIngress     *resources.FirewallRule
		FirewallDenyEgress      *resources.FirewallRule
		*resources.InstanceTemplate
		*resources.InstanceGroupManager
	}
}

func (t *Task) Create(ctx context.Context) error {
	logrus.Info("Creating resources...")
	steps := []common.Step{{
		Description: "Parsing PermissionSet...",
		Action:      t.DataSources.PermissionSet.Read,
	}, {
		Description: "Creating DefaultNetwork...",
		Action:      t.DataSources.DefaultNetwork.Read,
	}, {
		Description: "Reading Image...",
		Action:      t.DataSources.Image.Read,
	}, {
		Description: "Creating Bucket...",
		Action:      t.Resources.Bucket.Create,
	}, {
		Description: "Reading Credentials...",
		Action:      t.DataSources.Credentials.Read,
	}, {
		Description: "Creating FirewallInternalEgress...",
		Action:      t.Resources.FirewallInternalEgress.Create,
	}, {
		Description: "Creating FirewallInternalIngress...",
		Action:      t.Resources.FirewallInternalIngress.Create,
	}, {
		Description: "Creating FirewallExternalEgress...",
		Action:      t.Resources.FirewallExternalEgress.Create,
	}, {
		Description: "Creating FirewallExternalIngress...",
		Action:      t.Resources.FirewallExternalIngress.Create,
	}, {
		Description: "Creating FirewallDenyEgress...",
		Action:      t.Resources.FirewallDenyEgress.Create,
	}, {
		Description: "Creating FirewallDenyIngress...",
		Action:      t.Resources.FirewallDenyIngress.Create,
	}, {
		Description: "Creating InstanceTemplate...",
		Action:      t.Resources.InstanceTemplate.Create,
	}, {
		Description: "Creating InstanceGroupManager...",
		Action:      t.Resources.InstanceGroupManager.Create,
	}}

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
	logrus.Info("Creation completed")
	t.Attributes.Addresses = t.Resources.InstanceGroupManager.Attributes.Addresses
	t.Attributes.Status = t.Resources.InstanceGroupManager.Attributes.Status
	t.Attributes.Events = t.Resources.InstanceGroupManager.Attributes.Events
	return nil
}

func (t *Task) Read(ctx context.Context) error {
	logrus.Info("Reading resources... (this may happen several times)")
	steps := []common.Step{{
		Description: "Reading DefaultNetwork...",
		Action:      t.DataSources.DefaultNetwork.Read,
	}, {
		Description: "Reading Image...",
		Action:      t.DataSources.Image.Read,
	}, {
		Description: "Reading Bucket...",
		Action:      t.Resources.Bucket.Read,
	}, {
		Description: "Reading Credentials...",
		Action:      t.DataSources.Credentials.Read,
	}, {
		Description: "Reading FirewallInternalEgress...",
		Action:      t.Resources.FirewallInternalEgress.Read,
	}, {
		Description: "Reading FirewallInternalIngress...",
		Action:      t.Resources.FirewallInternalIngress.Read,
	}, {
		Description: "Reading FirewallExternalEgress...",
		Action:      t.Resources.FirewallExternalEgress.Read,
	}, {
		Description: "Reading FirewallExternalIngress...",
		Action:      t.Resources.FirewallExternalIngress.Read,
	}, {
		Description: "Reading FirewallDenyEgress...",
		Action:      t.Resources.FirewallDenyEgress.Read,
	}, {
		Description: "Reading FirewallDenyIngress...",
		Action:      t.Resources.FirewallDenyIngress.Read,
	}, {
		Description: "Reading InstanceTemplate...",
		Action:      t.Resources.InstanceTemplate.Read,
	}, {
		Description: "Reading InstanceGroupManager...",
		Action:      t.Resources.InstanceGroupManager.Read,
	}}
	if err := common.RunSteps(ctx, steps); err != nil {
		return err
	}
	logrus.Info("Read completed")
	t.Attributes.Addresses = t.Resources.InstanceGroupManager.Attributes.Addresses
	t.Attributes.Status = t.Resources.InstanceGroupManager.Attributes.Status
	t.Attributes.Events = t.Resources.InstanceGroupManager.Attributes.Events
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
					err := t.Pull(ctx)
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
		Description: "Deleting InstanceGroupManager...",
		Action:      t.Resources.InstanceGroupManager.Delete,
	}, {
		Description: "Deleting InstanceTemplate...",
		Action:      t.Resources.InstanceTemplate.Delete,
	}, {
		Description: "Deleting FirewallInternalEgress...",
		Action:      t.Resources.FirewallInternalEgress.Delete,
	}, {
		Description: "Deleting FirewallInternalIngress...",
		Action:      t.Resources.FirewallInternalIngress.Delete,
	}, {
		Description: "Deleting FirewallExternalEgress...",
		Action:      t.Resources.FirewallExternalEgress.Delete,
	}, {
		Description: "Deleting FirewallExternalIngress...",
		Action:      t.Resources.FirewallExternalIngress.Delete,
	}, {
		Description: "Deleting FirewallDenyEgress...",
		Action:      t.Resources.FirewallDenyEgress.Delete,
	}, {
		Description: "Deleting FirewallDenyIngress...",
		Action:      t.Resources.FirewallDenyIngress.Delete,
	}, {
		Description: "Deleting Bucket...",
		Action:      t.Resources.Bucket.Delete,
	}}...)
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

// Pull downloads the output directory from remote storage.
func (t *Task) Pull(ctx context.Context) error {
	src := t.DataSources.Credentials.Resource["RCLONE_REMOTE"] +
		filepath.Join("/data", t.Attributes.Environment.DirectoryOut)
	dst := filepath.Join(t.Attributes.Environment.Directory, t.Attributes.Environment.DirectoryOut)

	return machine.Transfer(ctx,
		src, dst,
		t.Attributes.Environment.ExcludeList,
	)
}

// Push uploads the work directory to remote storage.
func (t *Task) Push(ctx context.Context) error {
	// TODO remove
	for _, p := range t.Attributes.Environment.ExcludeList {
		logrus.Warnf("exclude pattern: %q", p)
	}
	return machine.Transfer(ctx,
		t.Attributes.Environment.Directory,
		t.DataSources.Credentials.Resource["RCLONE_REMOTE"]+"/data",
		t.Attributes.Environment.ExcludeList,
	)
}

func (t *Task) Start(ctx context.Context) error {
	return t.Resources.InstanceGroupManager.Update(ctx)
}

func (t *Task) Stop(ctx context.Context) error {
	original := t.Attributes.Parallelism
	defer func() { t.Attributes.Parallelism = original }()

	t.Attributes.Parallelism = 0
	return t.Resources.InstanceGroupManager.Update(ctx)
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
