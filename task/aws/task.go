package aws

import (
	"bytes"
	"context"
	"io"
	"log"
	"net"

	_ "github.com/rclone/rclone/backend/local"
	_ "github.com/rclone/rclone/backend/s3"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/sync"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/aws/resources"
	"terraform-provider-iterative/task/universal"
	"terraform-provider-iterative/task/universal/ssh"
)

func NewTask(ctx context.Context, cloud universal.Cloud, identifier string, task universal.Task) (*Task, error) {
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
	t.DataSources.DefaultVPCSubnet = resources.NewDefaultVPCSubnet(
		t.Client,
		t.DataSources.DefaultVPC,
	)
	t.DataSources.Image = resources.NewImage(
		t.Client,
		t.Attributes.Environment.Image,
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
		t.DataSources.Image,
		t.Resources.KeyPair,
		t.DataSources.Credentials,
		t.Attributes,
	)
	t.Resources.AutoScalingGroup = resources.NewAutoScalingGroup(
		t.Client,
		t.Identifier,
		t.DataSources.DefaultVPCSubnet,
		t.Resources.LaunchTemplate,
		t.Attributes.Parallelism,
		t.Attributes.Spot,
	)
	return t, nil
}

type Task struct {
	Client      *client.Client
	Identifier  string
	Attributes  universal.Task
	DataSources struct {
		*resources.DefaultVPC
		*resources.DefaultVPCSubnet
		*resources.Image
		*resources.Credentials
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
	log.Println("[INFO] Creating/Importing DefaultVPC...")
	if err := t.DataSources.DefaultVPC.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating/Importing DefaultVPCSubnet...")
	if err := t.DataSources.DefaultVPCSubnet.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating/Importing Image...")
	if err := t.DataSources.Image.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating/Importing Bucket...")
	if err := t.Resources.Bucket.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating/Importing SecurityGroup...")
	if err := t.Resources.SecurityGroup.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating/Importing KeyPair...")
	if err := t.Resources.KeyPair.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating/Importing Credentials...")
	if err := t.DataSources.Credentials.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating/Importing LaunchTemplate...")
	if err := t.Resources.LaunchTemplate.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Uploading/Refreshing Directory...")
	if t.Attributes.Environment.Directory != "" {
		if err := t.Push(ctx, t.Attributes.Environment.Directory, true); err != nil {
			return err
		}
	}
	log.Println("[INFO] Creating/Importing AutoScalingGroup...")
	if err := t.Resources.AutoScalingGroup.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Done!")
	t.Attributes.Addresses = t.Resources.AutoScalingGroup.Attributes.Addresses
	t.Attributes.Status = t.Resources.AutoScalingGroup.Attributes.Status
	t.Attributes.Events = t.Resources.AutoScalingGroup.Attributes.Events
	return nil
}

func (t *Task) Read(ctx context.Context) error {
	log.Println("[INFO] Reading DefaultVPC...")
	if err := t.DataSources.DefaultVPC.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading DefaultVPCSubnet...")
	if err := t.DataSources.DefaultVPCSubnet.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading Image...")
	if err := t.DataSources.Image.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading Bucket...")
	if err := t.Resources.Bucket.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading Credentials...")
	if err := t.DataSources.Credentials.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading SecurityGroup...")
	if err := t.Resources.SecurityGroup.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading KeyPair...")
	if err := t.Resources.KeyPair.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading Credentials...")
	if err := t.DataSources.Credentials.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading LaunchTemplate...")
	if err := t.Resources.LaunchTemplate.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading AutoScalingGroup...")
	if err := t.Resources.AutoScalingGroup.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Done!")
	t.Attributes.Addresses = t.Resources.AutoScalingGroup.Attributes.Addresses
	t.Attributes.Status = t.Resources.AutoScalingGroup.Attributes.Status
	t.Attributes.Events = t.Resources.AutoScalingGroup.Attributes.Events
	return nil
}

func (t *Task) Delete(ctx context.Context) error {
	log.Println("[INFO] Downloading Directory...")
	if t.Attributes.Environment.Directory != "" && t.Read(ctx) == nil {
		if err := t.Pull(ctx, t.Attributes.Environment.Directory); err != nil && err != universal.NotFoundError {
			return err
		}
	}
	log.Println("[INFO] Deleting AutoScalingGroup...")
	if err := t.Resources.AutoScalingGroup.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting LaunchTemplate...")
	if err := t.Resources.LaunchTemplate.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting KeyPair...")
	if err := t.Resources.KeyPair.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting SecurityGroup...")
	if err := t.Resources.SecurityGroup.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting Credentials...")
	if err := t.DataSources.Credentials.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting Bucket...")
	if err := t.Resources.Bucket.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting ")
	return nil
}

func (t *Task) Logs(ctx context.Context) ([]string, error) {
	if err := t.Read(ctx); err != nil {
		return nil, err
	}

	remote, err := fs.NewFs(ctx, (*t.DataSources.Credentials.Resource)["RCLONE_REMOTE"])
	if err != nil {
		return nil, err
	}

	entries, err := remote.List(ctx, "/log/task")
	if err != nil {
		return nil, err
	}

	var logs []string
	for _, entry := range entries {
		object, err := remote.NewObject(ctx, entry.Remote())
		if err != nil {
			return nil, err
		}
		reader, err := object.Open(ctx)
		if err != nil {
			return nil, err
		}
		buffer := new(bytes.Buffer)
		if _, err := io.Copy(buffer, reader); err != nil {
			return nil, err
		}
		logs = append(logs, buffer.String())
		reader.Close()
	}

	return logs, nil
}

func (t *Task) Pull(ctx context.Context, destination string) error {
	if err := t.Read(ctx); err != nil {
		return err
	}

	remote, err := fs.NewFs(ctx, (*t.DataSources.Credentials.Resource)["RCLONE_REMOTE"]+"/data")
	if err != nil {
		return err
	}

	local, err := fs.NewFs(ctx, destination)
	if err != nil {
		return err
	}

	if err := sync.CopyDir(ctx, local, remote, true); err != nil {
		return err
	}

	return nil
}

func (t *Task) Push(ctx context.Context, source string, unsafe bool) error {
	if err := t.Read(ctx); err != nil && !unsafe {
		return err
	}

	remote, err := fs.NewFs(ctx, (*t.DataSources.Credentials.Resource)["RCLONE_REMOTE"]+"/data")
	if err != nil {
		return err
	}

	local, err := fs.NewFs(ctx, source)
	if err != nil {
		return err
	}

	if err := sync.CopyDir(ctx, remote, local, true); err != nil {
		return err
	}

	return nil
}

func (t *Task) Stop(ctx context.Context) error {
	t.Attributes.Parallelism = 0
	return t.Resources.AutoScalingGroup.Update(ctx)
}

func (t *Task) GetAddresses(ctx context.Context) []net.IP {
	return t.Attributes.Addresses
}

func (t *Task) GetEvents(ctx context.Context) []universal.Event {
	return t.Attributes.Events
}

func (t *Task) GetStatus(ctx context.Context) map[string]int {
	return t.Attributes.Status
}

func (t *Task) GetKeyPair(ctx context.Context) (*ssh.DeterministicSSHKeyPair, error) {
	return t.Client.GetKeyPair(ctx)
}

func (t *Task) GetIdentifier(ctx context.Context) string {
	return t.Identifier
}
