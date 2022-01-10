package k8s

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/cp"

	_ "github.com/rclone/rclone/backend/local"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/sync"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/ssh"
	"terraform-provider-iterative/task/k8s/client"
	"terraform-provider-iterative/task/k8s/resources"
)

func New(ctx context.Context, cloud common.Cloud, identifier common.Identifier, task common.Task) (*Task, error) {
	client, err := client.New(ctx, cloud, task.Tags)
	if err != nil {
		return nil, err
	}

	persistentVolumeClaimStorageClass := ""
	persistentVolumeClaimSize := uint64(task.Size.Storage)
	persistentVolumeDirectory := task.Environment.Directory

	match := regexp.MustCompile(`^([^:]+):(?:(\d+):)?(.+)$`).FindStringSubmatch(task.Environment.Directory)
	if match != nil {
		persistentVolumeClaimStorageClass = match[1]
		if match[2] != "" {
			number, err := strconv.Atoi(match[2])
			if err != nil {
				return nil, err
			}
			persistentVolumeClaimSize = uint64(number)
		}
		persistentVolumeDirectory = match[3]
	}

	t := new(Task)
	t.Client = client
	t.Identifier = identifier
	t.Attributes.Task = task
	t.Attributes.Directory = persistentVolumeDirectory
	t.Attributes.DirectoryOut = persistentVolumeDirectory
	if task.Environment.DirectoryOut != "" {
		t.Attributes.DirectoryOut = task.Environment.DirectoryOut
	}

	t.Resources.ConfigMap = resources.NewConfigMap(
		t.Client,
		t.Identifier,
		map[string]string{"script": t.Attributes.Task.Environment.Script},
	)
	t.Resources.PersistentVolumeClaim = resources.NewPersistentVolumeClaim(
		t.Client,
		t.Identifier,
		persistentVolumeClaimStorageClass,
		persistentVolumeClaimSize,
		t.Attributes.Task.Parallelism > 1,
	)
	t.Resources.Job = resources.NewJob(
		t.Client,
		t.Identifier,
		t.Resources.PersistentVolumeClaim,
		t.Resources.ConfigMap,
		t.Attributes.Task,
	)
	return t, nil
}

type Task struct {
	Client     *client.Client
	Identifier common.Identifier
	Attributes struct {
		common.Task
		Directory    string
		DirectoryOut string
	}
	DataSources struct{}
	Resources   struct {
		*resources.ConfigMap
		*resources.PersistentVolumeClaim
		*resources.Job
	}
}

func (t *Task) Create(ctx context.Context) error {
	log.Println("[INFO] Creating ConfigMap...")
	if err := t.Resources.ConfigMap.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Creating PersistentVolumeClaim...")
	if err := t.Resources.PersistentVolumeClaim.Create(ctx); err != nil {
		return err
	}

	if t.Attributes.Directory != "" {
		os.Setenv("TPI_TRANSFER_MODE", "true")
		defer os.Unsetenv("TPI_TRANSFER_MODE")

		log.Println("[INFO] Deleting Job...")
		if err := t.Resources.Job.Delete(ctx); err != nil {
			return err
		}
		log.Println("[INFO] Creating ephemeral Job to upload directory...")
		if err := t.Resources.Job.Create(ctx); err != nil {
			return err
		}
		log.Println("[INFO] Uploading Directory...")
		if err := t.Push(ctx, t.Attributes.Directory); err != nil {
			return err
		}
		log.Println("[INFO] Deleting ephemeral Job to upload directory...")
		if err := t.Resources.Job.Delete(ctx); err != nil {
			return err
		}

		os.Unsetenv("TPI_TRANSFER_MODE")
	}

	log.Println("[INFO] Creating Job...")
	if err := t.Resources.Job.Create(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Done!")
	t.Attributes.Task.Addresses = t.Resources.Job.Attributes.Addresses
	t.Attributes.Task.Status = t.Resources.Job.Attributes.Status
	t.Attributes.Task.Events = t.Resources.Job.Attributes.Events
	return nil
}

func (t *Task) Read(ctx context.Context) error {
	log.Println("[INFO] Reading ConfigMap...")
	if err := t.Resources.ConfigMap.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading PersistentVolumeClaim...")
	if err := t.Resources.PersistentVolumeClaim.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Reading Job...")
	if err := t.Resources.Job.Read(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Done!")
	t.Attributes.Task.Addresses = t.Resources.Job.Attributes.Addresses
	t.Attributes.Task.Status = t.Resources.Job.Attributes.Status
	t.Attributes.Task.Events = t.Resources.Job.Attributes.Events
	return nil
}

func (t *Task) Delete(ctx context.Context) error {
	if t.Attributes.DirectoryOut != "" && t.Read(ctx) == nil {
		os.Setenv("TPI_TRANSFER_MODE", "true")
		os.Setenv("TPI_PULL_MODE", "true")
		defer os.Unsetenv("TPI_TRANSFER_MODE")
		defer os.Unsetenv("TPI_PULL_MODE")

		log.Println("[INFO] Deleting Job...")
		if err := t.Resources.Job.Delete(ctx); err != nil {
			return err
		}
		log.Println("[INFO] Creating ephemeral Job to retrieve directory...")
		if err := t.Resources.Job.Create(ctx); err != nil {
			return err
		}
		log.Println("[INFO] Downloading Directory...")
		if err := t.Pull(ctx, t.Attributes.DirectoryOut); err != nil {
			return err
		}

		log.Println("[INFO] Deleting ephemeral Job to retrieve directory...")
		if err := t.Resources.Job.Create(ctx); err != nil {
			return err
		}

		os.Unsetenv("TPI_TRANSFER_MODE")
		os.Unsetenv("TPI_PULL_MODE")
	}

	log.Println("[INFO] Deleting Job...")
	if err := t.Resources.Job.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting PersistentVolumeClaim...")
	if err := t.Resources.PersistentVolumeClaim.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Deleting ConfigMap...")
	if err := t.Resources.ConfigMap.Delete(ctx); err != nil {
		return err
	}
	log.Println("[INFO] Done!")
	return nil
}

func (t *Task) Push(ctx context.Context, source string) error {
	waitSelector := fmt.Sprintf("controller-uid=%s", t.Resources.Job.Resource.GetObjectMeta().GetLabels()["controller-uid"])
	pod, err := resources.WaitForPods(ctx, t.Client, 1*time.Second, t.Client.Cloud.Timeouts.Create, t.Client.Namespace, waitSelector)
	if err != nil {
		return err
	}

	ioStreams, _, _, _ := genericclioptions.NewTestIOStreams()
	copyOptions := cp.NewCopyOptions(ioStreams)
	copyOptions.Clientset = t.Client.ClientSet
	copyOptions.ClientConfig = t.Client.ClientConfig

	return copyOptions.Run([]string{source, fmt.Sprintf("%s/%s:%s", t.Client.Namespace, pod, "/directory/directory")})
}

func (t *Task) Pull(ctx context.Context, destination string) error {
	waitSelector := fmt.Sprintf("controller-uid=%s", t.Resources.Job.Resource.GetObjectMeta().GetLabels()["controller-uid"])
	pod, err := resources.WaitForPods(ctx, t.Client, 1*time.Second, t.Client.Cloud.Timeouts.Delete, t.Client.Namespace, waitSelector)
	if err != nil {
		return err
	}

	ioStreams, _, _, _ := genericclioptions.NewTestIOStreams()
	copyOptions := cp.NewCopyOptions(ioStreams)
	copyOptions.Clientset = t.Client.ClientSet
	copyOptions.ClientConfig = t.Client.ClientConfig

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	err = copyOptions.Run([]string{fmt.Sprintf("%s/%s:/directory/directory", t.Client.Namespace, pod), dir})
	if err != nil {
		return err
	}

	sourceFileSystem, err := fs.NewFs(ctx, dir)
	if err != nil {
		return err
	}

	destinationFileSystem, err := fs.NewFs(ctx, destination)
	if err != nil {
		return err
	}

	if err := sync.CopyDir(ctx, destinationFileSystem, sourceFileSystem, true); err != nil {
		return err
	}

	return nil
}

func (t *Task) Status(ctx context.Context) (common.Status, error) {
	return t.Attributes.Status, nil
}

func (t *Task) Events(ctx context.Context) []common.Event {
	return t.Attributes.Events
}

func (t *Task) Logs(ctx context.Context) ([]string, error) {
	return t.Resources.Job.Logs(ctx)
}

func (t *Task) Start(ctx context.Context) error {
	// FIXME: try experimental https://kubernetes.io/docs/concepts/workloads/controllers/job/#suspending-a-job
	return common.NotImplementedError
}

func (t *Task) Stop(ctx context.Context) error {
	// FIXME: try experimental https://kubernetes.io/docs/concepts/workloads/controllers/job/#suspending-a-job
	return common.NotImplementedError
}

func (t *Task) GetAddresses(ctx context.Context) []net.IP {
	return t.Attributes.Addresses
}

func (t *Task) GetKeyPair(ctx context.Context) (*ssh.DeterministicSSHKeyPair, error) {
	return nil, common.NotFoundError
}

func (t *Task) GetIdentifier(ctx context.Context) common.Identifier {
	return t.Identifier
}
