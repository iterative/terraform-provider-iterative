package resources

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	terraform_resource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	kubernetes_batch "k8s.io/api/batch/v1"
	kubernetes_core "k8s.io/api/core/v1"
	kubernetes_errors "k8s.io/apimachinery/pkg/api/errors"
	kubernetes_resource "k8s.io/apimachinery/pkg/api/resource"
	kubernetes_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/k8s/client"
)

func NewJob(client *client.Client, identifier common.Identifier, persistentVolumeClaim *PersistentVolumeClaim, configMap *ConfigMap, task common.Task) *Job {
	j := new(Job)
	j.Client = client
	j.Identifier = identifier.Long()
	j.Dependencies.PersistentVolumeClaim = persistentVolumeClaim
	j.Dependencies.ConfigMap = configMap
	j.Attributes.Task = task
	j.Attributes.Parallelism = task.Parallelism
	return j
}

type Job struct {
	Client     *client.Client
	Identifier string
	Attributes struct {
		Task        common.Task
		Parallelism uint16
		Addresses   []net.IP
		Status      common.Status
		Events      []common.Event
	}
	Dependencies struct {
		*PersistentVolumeClaim
		*ConfigMap
	}
	Resource *kubernetes_batch.Job
}

func (j *Job) Create(ctx context.Context) error {
	size := j.Attributes.Task.Size.Machine
	sizes := map[string]string{
		"m":       "8-32000",
		"l":       "32-128000",
		"xl":      "64-256000",
		"m+k80":   "4-64000+nvidia-tesla-k80*1",
		"l+k80":   "32-512000+nvidia-tesla-k80*8",
		"xl+k80":  "64-768000+nvidia-tesla-k80*16",
		"m+v100":  "8-64000+nvidia-tesla-v100*1",
		"l+v100":  "32-256000+nvidia-tesla-v100*4",
		"xl+v100": "64-512000+nvidia-tesla-v100*8",
	}

	if val, ok := sizes[size]; ok {
		size = val
	}

	match := regexp.MustCompile(`^(\d+)-(\d+)(?:\+([^*]+)\*([1-9]\d*))?$`).FindStringSubmatch(size)
	if match == nil {
		return common.NotFoundError
	}

	// Define the accelerator settings (i.e. GPU type, model, ...)
	jobNodeSelector := map[string]string{}
	jobAccelerator := match[3]
	jobGPUType := "kubernetes.io/gpu"
	jobGPUCount := match[4]

	// Define the dynamic resource allocation limits for the job pods.
	jobLimits := kubernetes_core.ResourceList{}
	jobLimits[kubernetes_core.ResourceMemory] = kubernetes_resource.MustParse(match[2] + "M")
	jobLimits[kubernetes_core.ResourceCPU] = kubernetes_resource.MustParse(match[1])
	if diskAmount := j.Attributes.Task.Size.Storage; diskAmount > 0 {
		jobLimits[kubernetes_core.ResourceEphemeralStorage] = kubernetes_resource.MustParse(strconv.Itoa(diskAmount) + "G")
	}

	// If the resource requires GPU provisioning, determine how many GPUs and the kind of GPU it needs.
	if jobGPUCount > "0" {
		jobLimits[kubernetes_core.ResourceName(jobGPUType)] = kubernetes_resource.MustParse(jobGPUCount)
		if jobAccelerator != "" {
			jobNodeSelector = map[string]string{"accelerator": jobAccelerator}
		}
	}

	// Leave the job running for 30 seconds after the termination signal
	jobTerminationGracePeriod := int64(30)

	jobBackoffLimit := int32(math.MaxInt32)
	jobCompletions := int32(j.Attributes.Task.Parallelism)
	jobParallelism := int32(j.Attributes.Task.Parallelism)

	jobActiveDeadlineSeconds := int64(j.Attributes.Task.Environment.Timeout / time.Second)

	jobEnvironment := []kubernetes_core.EnvVar{}
	for name, value := range j.Attributes.Task.Environment.Variables.Enrich() {
		jobEnvironment = append(jobEnvironment, kubernetes_core.EnvVar{
			Name:  name,
			Value: value,
		})
	}
	jobEnvironment = append(jobEnvironment, kubernetes_core.EnvVar{
		Name:  "TPI_TRANSFER_MODE",
		Value: os.Getenv("TPI_TRANSFER_MODE"),
	})
	jobEnvironment = append(jobEnvironment, kubernetes_core.EnvVar{
		Name:  "TPI_PULL_MODE",
		Value: os.Getenv("TPI_PULL_MODE"),
	})

	readExecuteUserGroupOthers := int32(0555)

	jobVolumes := []kubernetes_core.Volume{
		{
			Name: j.Identifier + "-cm",
			VolumeSource: kubernetes_core.VolumeSource{
				ConfigMap: &kubernetes_core.ConfigMapVolumeSource{
					LocalObjectReference: kubernetes_core.LocalObjectReference{
						Name: j.Dependencies.ConfigMap.Identifier,
					},
					DefaultMode: &readExecuteUserGroupOthers,
				},
			},
		},
	}

	jobVolumeMounts := []kubernetes_core.VolumeMount{
		{
			Name:      j.Identifier + "-cm",
			MountPath: "/script",
		},
	}

	if j.Attributes.Task.Environment.Directory != "" {
		jobVolumeMounts = append(jobVolumeMounts, kubernetes_core.VolumeMount{
			Name:      j.Identifier + "-pvc",
			MountPath: "/directory",
		})
		jobVolumes = append(jobVolumes, kubernetes_core.Volume{
			Name: j.Identifier + "-pvc",
			VolumeSource: kubernetes_core.VolumeSource{
				PersistentVolumeClaim: &kubernetes_core.PersistentVolumeClaimVolumeSource{
					ClaimName: j.Dependencies.PersistentVolumeClaim.Identifier,
				},
			},
		})
	}

	// Running with /bin/sh -c as the ENTRYPOINT, this script will be in charge of allowing
	// seamless data synchronization. The first branch of the conditional will run on destroy
	// allowing the provider to manage the pod lifecycle without letting it exit on "completion".
	// The second branch will run on apply, waiting for the file copy to complete before starting
	// the script.
	script := `
	if ! test -z "$TPI_TRANSFER_MODE"; then
	  test -z "$TPI_PULL_MODE" && rm -r /directory/directory
	  while true; do
	    sleep 86400
	  done
	else
	  cd /directory/directory
	  exec /script/script
	fi
	`

	job := kubernetes_batch.Job{
		ObjectMeta: kubernetes_meta.ObjectMeta{
			Name:        j.Identifier,
			Namespace:   j.Client.Namespace,
			Labels:      j.Client.Tags,
			Annotations: j.Client.Tags,
		},
		Spec: kubernetes_batch.JobSpec{
			ActiveDeadlineSeconds: &jobActiveDeadlineSeconds,
			BackoffLimit:          &jobBackoffLimit,
			Completions:           &jobCompletions,
			Parallelism:           &jobParallelism,
			// We don't want jobs to delete themselves upon completion, because
			// that would also mean losing logs before users check them.
			// TTLSecondsAfterFinished: &jobTTLSecondsAfterFinished,
			Template: kubernetes_core.PodTemplateSpec{
				Spec: kubernetes_core.PodSpec{
					TerminationGracePeriodSeconds: &jobTerminationGracePeriod,
					ActiveDeadlineSeconds:         &jobActiveDeadlineSeconds,
					// We don't want pods to restart if the container exits with a non–zero status.
					// Only when there is a pod failure.
					RestartPolicy: kubernetes_core.RestartPolicyNever,
					NodeSelector:  jobNodeSelector,
					Containers: []kubernetes_core.Container{
						{
							Name:  j.Identifier,
							Image: j.Attributes.Task.Environment.Image,
							Resources: kubernetes_core.ResourceRequirements{
								Limits: jobLimits,
								Requests: kubernetes_core.ResourceList{
									// Don't allocate any resources statically and let the pod scale vertically when and if required.
									kubernetes_core.ResourceMemory: kubernetes_resource.MustParse("0"),
									kubernetes_core.ResourceCPU:    kubernetes_resource.MustParse("0"),
								},
							},
							Command: []string{
								"sh", "-c", script,
							},
							Env:          jobEnvironment,
							VolumeMounts: jobVolumeMounts,
						},
					},
					Volumes: jobVolumes,
				},
			},
		},
	}

	// Ask Kubernetes to create the job.
	out, err := j.Client.Services.Batch.Jobs(j.Client.Namespace).Create(ctx, &job, kubernetes_meta.CreateOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 409 {
			return j.Read(ctx)
		}
		return err
	}

	j.Resource = out
	return nil
}

func (j *Job) Read(ctx context.Context) error {
	job, err := j.Client.Services.Batch.Jobs(j.Client.Namespace).Get(ctx, j.Identifier, kubernetes_meta.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return common.NotFoundError
		}
		return err
	}
	eventListOptions := kubernetes_meta.ListOptions{FieldSelector: fields.OneTermEqualSelector("involvedObject.name", job.Name).String()}
	events, err := j.Client.Services.Core.Events(j.Client.Namespace).List(ctx, eventListOptions)
	if err != nil {
		return err
	}
	for _, event := range events.Items {
		j.Attributes.Events = append(j.Attributes.Events, common.Event{
			Time: event.FirstTimestamp.Time,
			Code: event.Message,
			Description: []string{
				event.Reason,
				event.Action,
			},
		})
	}
	j.Attributes.Status = common.Status{
		common.StatusCodeActive:    int(job.Status.Active),
		common.StatusCodeSucceeded: int(job.Status.Succeeded),
		common.StatusCodeFailed:    int(job.Status.Failed),
	}
	j.Resource = job
	return nil
}

func (j *Job) Delete(ctx context.Context) error {
	_, err := j.Client.Services.Batch.Jobs(j.Client.Namespace).Get(ctx, j.Identifier, kubernetes_meta.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return nil
		}
		return err
	}

	// DeletePropagationForeground deletes the resources and causes the garbage
	// collector to delete dependent resources and wait for all dependents whose
	// ownerReference.blockOwnerDeletion=true.
	propagationPolicy := kubernetes_meta.DeletePropagationForeground

	err = j.Client.Services.Batch.Jobs(j.Client.Namespace).Delete(ctx, j.Identifier, kubernetes_meta.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil {
		return fmt.Errorf("Failed to delete Job! API error: %s", err)
	}

	err = terraform_resource.RetryContext(ctx, j.Client.Cloud.Timeouts.Delete, func() *terraform_resource.RetryError {
		_, err := j.Client.Services.Batch.Jobs(j.Client.Namespace).Get(ctx, j.Identifier, kubernetes_meta.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
				return nil
			}
			return terraform_resource.NonRetryableError(err)
		}

		e := fmt.Errorf("Job %s still exists", j.Identifier)
		return terraform_resource.RetryableError(e)
	})
	if err != nil {
		return err
	}

	return nil
}

func (j *Job) Logs(ctx context.Context) ([]string, error) {
	pods, err := j.Client.Services.Core.Pods(j.Client.Namespace).List(ctx, kubernetes_meta.ListOptions{
		LabelSelector: fmt.Sprintf("controller-uid=%s", j.Resource.GetObjectMeta().GetLabels()["controller-uid"]),
	})
	if err != nil {
		return nil, err
	}

	var result []string

	for _, pod := range pods.Items {
		logs, err := j.Client.Services.Core.Pods(j.Client.Namespace).GetLogs(pod.Name, &kubernetes_core.PodLogOptions{}).Stream(ctx)
		if err != nil {
			if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && strings.HasSuffix(statusErr.ErrStatus.Message, "ContainerCreating") {
				continue
			}
			return nil, err
		}
		defer logs.Close()

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, logs)
		if err != nil {
			return nil, err
		}

		result = append(result, buf.String())
	}

	return result, nil
}
