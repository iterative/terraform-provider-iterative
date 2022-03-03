package gcp

import (
	"context"
	"log"
	"net"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
	"terraform-provider-iterative/task/common/ssh"
	"terraform-provider-iterative/task/gcp/client"
	"terraform-provider-iterative/task/gcp/resources"
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
	log.Println("[DEBUG] Creating DefaultNetwork...")
	if err := t.DataSources.DefaultNetwork.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Creating Image...")
	if err := t.DataSources.Image.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Creating Bucket...")
	if err := t.Resources.Bucket.Create(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Creating Credentials...")
	if err := t.DataSources.Credentials.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Creating FirewallInternalEgress...")
	if err := t.Resources.FirewallInternalEgress.Create(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Creating FirewallInternalIngress...")
	if err := t.Resources.FirewallInternalIngress.Create(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Creating FirewallExternalEgress...")
	if err := t.Resources.FirewallExternalEgress.Create(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Creating FirewallExternalIngress...")
	if err := t.Resources.FirewallExternalIngress.Create(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Creating FirewallDenyEgress...")
	if err := t.Resources.FirewallDenyEgress.Create(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Creating FirewallDenyIngress...")
	if err := t.Resources.FirewallDenyIngress.Create(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Creating InstanceTemplate...")
	if err := t.Resources.InstanceTemplate.Create(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Creating InstanceGroupManager...")
	if err := t.Resources.InstanceGroupManager.Create(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Uploading Directory...")
	if t.Attributes.Environment.Directory != "" {
		if err := t.Push(ctx, t.Attributes.Environment.Directory); err != nil {
			return err
		}
	}
	log.Println("[DEBUG] Starting task...")
	if err := t.Start(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Done!")
	t.Attributes.Addresses = t.Resources.InstanceGroupManager.Attributes.Addresses
	t.Attributes.Status = t.Resources.InstanceGroupManager.Attributes.Status
	t.Attributes.Events = t.Resources.InstanceGroupManager.Attributes.Events
	return nil
}

func (t *Task) Read(ctx context.Context) error {
	log.Println("[DEBUG] Reading DefaultNetwork...")
	if err := t.DataSources.DefaultNetwork.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Reading Image...")
	if err := t.DataSources.Image.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Reading Bucket...")
	if err := t.Resources.Bucket.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Reading Credentials...")
	if err := t.DataSources.Credentials.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Reading FirewallInternalEgress...")
	if err := t.Resources.FirewallInternalEgress.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Reading FirewallInternalIngress...")
	if err := t.Resources.FirewallInternalIngress.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Reading FirewallExternalEgress...")
	if err := t.Resources.FirewallExternalEgress.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Reading FirewallExternalIngress...")
	if err := t.Resources.FirewallExternalIngress.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Reading FirewallDenyEgress...")
	if err := t.Resources.FirewallDenyEgress.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Reading FirewallDenyIngress...")
	if err := t.Resources.FirewallDenyIngress.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Reading InstanceTemplate...")
	if err := t.Resources.InstanceTemplate.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Reading InstanceGroupManager...")
	if err := t.Resources.InstanceGroupManager.Read(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Done!")
	t.Attributes.Addresses = t.Resources.InstanceGroupManager.Attributes.Addresses
	t.Attributes.Status = t.Resources.InstanceGroupManager.Attributes.Status
	t.Attributes.Events = t.Resources.InstanceGroupManager.Attributes.Events
	return nil
}

func (t *Task) Delete(ctx context.Context) error {
	log.Println("[DEBUG] Downloading Directory...")
	if t.Attributes.Environment.DirectoryOut != "" && t.Read(ctx) == nil {
		if err := t.Pull(ctx, t.Attributes.Environment.DirectoryOut); err != nil && err != common.NotFoundError {
			return err
		}
	}
	log.Println("[DEBUG] Deleting InstanceGroupManager...")
	if err := t.Resources.InstanceGroupManager.Delete(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Deleting InstanceTemplate...")
	if err := t.Resources.InstanceTemplate.Delete(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Deleting FirewallInternalEgress...")
	if err := t.Resources.FirewallInternalEgress.Delete(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Deleting FirewallInternalIngress...")
	if err := t.Resources.FirewallInternalIngress.Delete(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Deleting FirewallExternalEgress...")
	if err := t.Resources.FirewallExternalEgress.Delete(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Deleting FirewallExternalIngress...")
	if err := t.Resources.FirewallExternalIngress.Delete(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Deleting FirewallDenyEgress...")
	if err := t.Resources.FirewallDenyEgress.Delete(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Deleting FirewallDenyIngress...")
	if err := t.Resources.FirewallDenyIngress.Delete(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Deleting Bucket...")
	if err := t.Resources.Bucket.Delete(ctx); err != nil {
		return err
	}
	log.Println("[DEBUG] Done!")
	return nil
}

func (t *Task) Logs(ctx context.Context) ([]string, error) {
	if err := t.Read(ctx); err != nil {
		return nil, err
	}

	return machine.Logs(ctx, (*t.DataSources.Credentials.Resource)["RCLONE_REMOTE"])
}

func (t *Task) Pull(ctx context.Context, destination string) error {
	if err := t.Read(ctx); err != nil {
		return err
	}

	return machine.Transfer(ctx, (*t.DataSources.Credentials.Resource)["RCLONE_REMOTE"]+"/data", destination)
}

func (t *Task) Push(ctx context.Context, source string) error {
	if err := t.Read(ctx); err != nil {
		return err
	}

	return machine.Transfer(ctx, source, (*t.DataSources.Credentials.Resource)["RCLONE_REMOTE"]+"/data")
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

	return machine.Status(ctx, (*t.DataSources.Credentials.Resource)["RCLONE_REMOTE"], t.Attributes.Status)
}

func (t *Task) GetKeyPair(ctx context.Context) (*ssh.DeterministicSSHKeyPair, error) {
	return t.Client.GetKeyPair(ctx)
}

func (t *Task) GetIdentifier(ctx context.Context) common.Identifier {
	return t.Identifier
}
