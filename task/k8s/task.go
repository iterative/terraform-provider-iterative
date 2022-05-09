package k8s

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/cp"

	_ "github.com/rclone/rclone/backend/local"

	"github.com/sirupsen/logrus"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
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
	persistentVolumeClaimSize := task.Size.Storage
	persistentVolumeDirectory := task.Environment.Directory

	match := regexp.MustCompile(`^([^:]+):(?:(\d+):)?(.+)$`).FindStringSubmatch(task.Environment.Directory)
	if match != nil {
		persistentVolumeClaimStorageClass = match[1]
		if match[2] != "" {
			number, err := strconv.Atoi(match[2])
			if err != nil {
				return nil, err
			}
			persistentVolumeClaimSize = int(number)
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
	logrus.Info("Creating resources...")
	logrus.Info("[1/7] Creating ConfigMap...")
	if err := t.Resources.ConfigMap.Create(ctx); err != nil {
		return err
	}
	logrus.Info("[2/7] Creating PersistentVolumeClaim...")
	if err := t.Resources.PersistentVolumeClaim.Create(ctx); err != nil {
		return err
	}

	if t.Attributes.Directory != "" {
		os.Setenv("TPI_TRANSFER_MODE", "true")
		defer os.Unsetenv("TPI_TRANSFER_MODE")

		logrus.Info("[3/7] Deleting Job...")
		if err := t.Resources.Job.Delete(ctx); err != nil {
			return err
		}
		logrus.Info("[4/7] Creating ephemeral Job to upload directory...")
		if err := t.Resources.Job.Create(ctx); err != nil {
			return err
		}
		logrus.Info("[5/7] Uploading Directory...")
		if err := t.Push(ctx, t.Attributes.Directory); err != nil {
			return err
		}
		logrus.Info("[6/7] Deleting ephemeral Job to upload directory...")
		if err := t.Resources.Job.Delete(ctx); err != nil {
			return err
		}

		os.Unsetenv("TPI_TRANSFER_MODE")
	}

	logrus.Info("[7/7] Creating Job...")
	if err := t.Resources.Job.Create(ctx); err != nil {
		return err
	}
	logrus.Info("Creation completed")
	t.Attributes.Task.Addresses = t.Resources.Job.Attributes.Addresses
	t.Attributes.Task.Status = t.Resources.Job.Attributes.Status
	t.Attributes.Task.Events = t.Resources.Job.Attributes.Events
	return nil
}

func (t *Task) Read(ctx context.Context) error {
	logrus.Info("Reading resources... (this may happen several times)")
	logrus.Info("[1/3] Reading ConfigMap...")
	if err := t.Resources.ConfigMap.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[2/3] Reading PersistentVolumeClaim...")
	if err := t.Resources.PersistentVolumeClaim.Read(ctx); err != nil {
		return err
	}
	logrus.Info("[3/3] Reading Job...")
	if err := t.Resources.Job.Read(ctx); err != nil {
		return err
	}
	logrus.Info("Read completed")
	t.Attributes.Task.Addresses = t.Resources.Job.Attributes.Addresses
	t.Attributes.Task.Status = t.Resources.Job.Attributes.Status
	t.Attributes.Task.Events = t.Resources.Job.Attributes.Events
	return nil
}

func (t *Task) Delete(ctx context.Context) error {
	logrus.Info("Deleting resources...")
	if t.Attributes.DirectoryOut != "" && t.Read(ctx) == nil {
		os.Setenv("TPI_TRANSFER_MODE", "true")
		os.Setenv("TPI_PULL_MODE", "true")
		defer os.Unsetenv("TPI_TRANSFER_MODE")
		defer os.Unsetenv("TPI_PULL_MODE")

		logrus.Info("[1/7] Deleting Job...")
		if err := t.Resources.Job.Delete(ctx); err != nil {
			return err
		}
		logrus.Info("[2/7] Creating ephemeral Job to retrieve directory...")
		if err := t.Resources.Job.Create(ctx); err != nil {
			return err
		}
		logrus.Info("[3/7] Downloading Directory...")
		if err := t.Pull(ctx, t.Attributes.Directory, t.Attributes.DirectoryOut); err != nil {
			return err
		}

		logrus.Info("[4/7] Deleting ephemeral Job to retrieve directory...")
		if err := t.Resources.Job.Create(ctx); err != nil {
			return err
		}

		os.Unsetenv("TPI_TRANSFER_MODE")
		os.Unsetenv("TPI_PULL_MODE")
	}

	logrus.Info("[5/7] Deleting Job...")
	if err := t.Resources.Job.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("[6/7] Deleting PersistentVolumeClaim...")
	if err := t.Resources.PersistentVolumeClaim.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("[7/7] Deleting ConfigMap...")
	if err := t.Resources.ConfigMap.Delete(ctx); err != nil {
		return err
	}
	logrus.Info("Deletion completed")
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

func (t *Task) Pull(ctx context.Context, destination, include string) error {
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

	return machine.Transfer(ctx, dir, destination, include)
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
	return nil, common.NotImplementedError
}

func (t *Task) GetIdentifier(ctx context.Context) common.Identifier {
	return t.Identifier
}
