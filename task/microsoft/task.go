package microsoft

import (
	"bytes"
	"context"
	"io"
	"log"
	"net"

	_ "github.com/rclone/rclone/backend/azureblob"
	_ "github.com/rclone/rclone/backend/local"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/sync"

	"terraform-provider-iterative/task/microsoft/client"
	"terraform-provider-iterative/task/microsoft/resources"
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
		t.Attributes,
	)
	return t, nil
}

type Task struct {
	Client      *client.Client
	Identifier  string
	Attributes  universal.Task
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
	log.Println("[INFO] Creating ResourceGroup...")
	if err := t.Resources.ResourceGroup.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating StorageAccount...")
	if err := t.Resources.StorageAccount.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating BlobContainer...")
	if err := t.Resources.BlobContainer.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating Credentials...")
	if err := t.DataSources.Credentials.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating VirtualNetwork...")
	if err := t.Resources.VirtualNetwork.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating SecurityGroup...")
	if err := t.Resources.SecurityGroup.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating Subnet...")
	if err := t.Resources.Subnet.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Uploading Directory...")
	if t.Attributes.Environment.Directory != "" {
		if err := t.Push(ctx, t.Attributes.Environment.Directory, true); err != nil {
			return err
		}
	}
	log.Println("[INFO] Creating VirtualMachineScaleSet...")
	if err := t.Resources.VirtualMachineScaleSet.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Done!")
	t.Attributes.Addresses = t.Resources.VirtualMachineScaleSet.Attributes.Addresses
	t.Attributes.Status = t.Resources.VirtualMachineScaleSet.Attributes.Status
	t.Attributes.Events = t.Resources.VirtualMachineScaleSet.Attributes.Events
	return nil
}

func (t *Task) Read(ctx context.Context) error {
	log.Println("[INFO] Reading ResourceGroup...")
	if err := t.Resources.ResourceGroup.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading StorageAccount...")
	if err := t.Resources.StorageAccount.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading BlobContainer...")
	if err := t.Resources.BlobContainer.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading Credentials...")
	if err := t.DataSources.Credentials.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading VirtualNetwork...")
	if err := t.Resources.VirtualNetwork.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading SecurityGroup...")
	if err := t.Resources.SecurityGroup.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading Subnet...")
	if err := t.Resources.Subnet.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading VirtualMachineScaleSet...")
	if err := t.Resources.VirtualMachineScaleSet.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Done!")
	t.Attributes.Addresses = t.Resources.VirtualMachineScaleSet.Attributes.Addresses
	t.Attributes.Status = t.Resources.VirtualMachineScaleSet.Attributes.Status
	t.Attributes.Events = t.Resources.VirtualMachineScaleSet.Attributes.Events
	return nil
}

func (t *Task) Delete(ctx context.Context) error {
	log.Println("[INFO] Downloading Directory...")
	if t.Attributes.Environment.Directory != "" && t.Read(ctx) == nil {
		if err := t.Pull(ctx, t.Attributes.Environment.Directory); err != nil && err != universal.NotFoundError {
			return err
		}
	}
	log.Println("[INFO] Deleting VirtualMachineScaleSet...")
	if err := t.Resources.VirtualMachineScaleSet.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting Subnet...")
	if err := t.Resources.Subnet.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting SecurityGroup...")
	if err := t.Resources.SecurityGroup.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting VirtualNetwork...")
	if err := t.Resources.VirtualNetwork.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting BlobContainer...")
	if err := t.Resources.BlobContainer.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting StorageAccount...")
	if err := t.Resources.StorageAccount.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting ResourceGroup...")
	if err := t.Resources.ResourceGroup.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Done!")
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
	return t.Resources.VirtualMachineScaleSet.Update(ctx)
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
