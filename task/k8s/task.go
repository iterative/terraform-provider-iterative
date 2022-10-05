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

func List(ctx context.Context, cloud common.Cloud) ([]common.Identifier, error) {
	client, err := client.New(ctx, cloud, nil)
	if err != nil {
		return nil, err
	}

	return resources.ListConfigMaps(ctx, client)
}

func New(ctx context.Context, cloud common.Cloud, identifier common.Identifier, task common.Task) (*Task, error) {
	client, err := client.New(ctx, cloud, cloud.Tags)
	if err != nil {
		return nil, err
	}

	t := new(Task)
	t.Client = client
	t.Identifier = identifier
	t.Attributes.Task = task
	t.Attributes.Directory = task.Environment.Directory
	t.Attributes.DirectoryOut = task.Environment.Directory
	if task.Environment.DirectoryOut != "" {
		t.Attributes.DirectoryOut = task.Environment.DirectoryOut
	}

	t.DataSources.PermissionSet = resources.NewPermissionSet(
		t.Client,
		t.Attributes.Task.PermissionSet,
	)
	t.Resources.ConfigMap = resources.NewConfigMap(
		t.Client,
		t.Identifier,
		map[string]string{"script": t.Attributes.Task.Environment.Script},
	)
	var pvc resources.VolumeInfoProvider
	if task.RemoteStorage != nil {
		t.DataSources.ExistingPersistentVolumeClaim = resources.NewExistingPersistentVolumeClaim(
			t.Client, *task.RemoteStorage)
		pvc = t.DataSources.ExistingPersistentVolumeClaim
	} else {
		var persistentVolumeDirectory string
		var persistentVolumeClaimStorageClass string
		persistentVolumeClaimSize := task.Size.Storage

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
			t.Attributes.Directory = persistentVolumeDirectory
		}

		t.Resources.PersistentVolumeClaim = resources.NewPersistentVolumeClaim(
			t.Client,
			t.Identifier,
			persistentVolumeClaimStorageClass,
			persistentVolumeClaimSize,
			t.Attributes.Task.Parallelism > 1,
		)
		pvc = t.Resources.PersistentVolumeClaim
	}
	t.Resources.Job = resources.NewJob(
		t.Client,
		t.Identifier,
		pvc,
		t.Resources.ConfigMap,
		t.DataSources.PermissionSet,
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
	DataSources struct {
		PermissionSet                 *resources.PermissionSet
		ExistingPersistentVolumeClaim *resources.ExistingPersistentVolumeClaim
	}
	Resources struct {
		ConfigMap             *resources.ConfigMap
		PersistentVolumeClaim *resources.PersistentVolumeClaim
		Job                   *resources.Job
	}
}

func (t *Task) Create(ctx context.Context) error {
	logrus.Info("Creating resources...")
	steps := []common.Step{{
		Description: "Parsing PermissionSet...",
		Action:      t.DataSources.PermissionSet.Read,
	}, {
		Description: "Creating ConfigMap...",
		Action:      t.Resources.ConfigMap.Create,
	}}
	if t.Resources.PersistentVolumeClaim != nil {
		steps = append(steps, common.Step{
			Description: "Creating PersistentVolumeClaim...",
			Action:      t.Resources.PersistentVolumeClaim.Create,
		})
	}

	if t.Attributes.Directory != "" {
		env := map[string]string{
			"TPI_TRANSFER_MODE": "true",
		}
		steps = append(steps, []common.Step{{
			Description: "Deleting Job...",
			Action:      withEnv(env, t.Resources.Job.Delete),
		}, {
			Description: "Creating ephemeral Job to upload directory...",
			Action:      withEnv(env, t.Resources.Job.Create),
		}, {
			Description: "Uploading Directory...",
			Action: withEnv(env, func(ctx context.Context) error {
				return t.Push(ctx, t.Attributes.Directory)
			}),
		}, {
			Description: "Deleting ephemeral Job to upload directory...",
			Action:      withEnv(env, t.Resources.Job.Delete),
		}}...)
	}

	steps = append(steps, common.Step{
		Description: "Creating Job...",
		Action:      t.Resources.Job.Create,
	})
	if err := common.RunSteps(ctx, steps); err != nil {
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
	steps := []common.Step{{
		Description: "Reading ConfigMap...",
		Action:      t.Resources.ConfigMap.Read,
	}, {
		Description: "Reading PersistentVolumeClaim...",
		Action: func(ctx context.Context) error {
			if t.Resources.PersistentVolumeClaim != nil {
				return t.Resources.PersistentVolumeClaim.Read(ctx)
			} else if t.DataSources.ExistingPersistentVolumeClaim != nil {
				return t.DataSources.ExistingPersistentVolumeClaim.Read(ctx)
			}
			return fmt.Errorf("misconfigured storage")
		},
	}, {
		Description: "Reading Job...",
		Action:      t.Resources.Job.Read,
	}}
	if err := common.RunSteps(ctx, steps); err != nil {
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
	steps := []common.Step{}
	if t.Attributes.DirectoryOut != "" && t.Read(ctx) == nil {
		env := map[string]string{
			"TPI_TRANSFER_MODE": "true",
			"TPI_PULL_MODE":     "true",
		}

		steps = []common.Step{{
			Description: "Deleting Job...",
			Action:      withEnv(env, t.Resources.Job.Delete),
		}, {
			Description: "Creating ephemeral Job to retrieve directory...",
			Action:      withEnv(env, t.Resources.Job.Create),
		}, {
			Description: "Downloading Directory...",
			Action: withEnv(env, func(ctx context.Context) error {
				return t.Pull(ctx, t.Attributes.Directory, t.Attributes.DirectoryOut)
			}),
		}, {
			// WTH?
			Description: "Deleting ephemeral Job to retrieve directory...",
			Action:      withEnv(env, t.Resources.Job.Create),
		}}
	}

	steps = append(steps, []common.Step{{
		Description: "Deleting Job...",
		Action:      t.Resources.Job.Delete,
	}, {
		Description: "Deleting ConfigMap...",
		Action:      t.Resources.ConfigMap.Delete,
	}}...)
	if t.Resources.PersistentVolumeClaim != nil {
		steps = append(steps, common.Step{
			Description: "Deleting PersistentVolumeClaim...",
			Action:      t.Resources.PersistentVolumeClaim.Delete,
		})
	}

	if err := common.RunSteps(ctx, steps); err != nil {
		return err
	}
	logrus.Info("Deletion completed")
	return nil
}

func (t *Task) Push(ctx context.Context, source string) error {
	waitSelector := fmt.Sprintf("controller-uid=%s", t.Resources.Job.Resource.Spec.Selector.MatchLabels["controller-uid"])
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
	waitSelector := fmt.Sprintf("controller-uid=%s", t.Resources.Job.Resource.Spec.Selector.MatchLabels["controller-uid"])
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

// withEnv runs the specified action with the environment variables set.
func withEnv(env map[string]string, action func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		for key, value := range env {
			os.Setenv(key, value)
			defer os.Unsetenv(key)
		}
		return action(ctx)
	}
}
