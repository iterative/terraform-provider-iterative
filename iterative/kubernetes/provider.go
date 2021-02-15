package kubernetes

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	terraform_resource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	terraform_schema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	kubernetes_batch "k8s.io/api/batch/v1"
	kubernetes_core "k8s.io/api/core/v1"
	kubernetes_errors "k8s.io/apimachinery/pkg/api/errors"
	kubernetes_resource "k8s.io/apimachinery/pkg/api/resource"
	kubernetes_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes_wait "k8s.io/apimachinery/pkg/util/wait"
	kubernetes "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	kubernetes_clientcmd "k8s.io/client-go/tools/clientcmd"
)

func ResourceMachineCreate(ctx context.Context, d *terraform_schema.ResourceData, meta interface{}) error {
	conn, err := kubernetesClient()
	if err != nil {
		return err
	}
	instanceType, err := getInstanceType(d.Get("instance_type").(string), d.Get("instance_gpu").(string))
	if err != nil {
		return err
	}
	jobNamespace := "iterative"
	jobReadinessCommand := d.Get("kubernetes_readiness_command").(string)
	jobNodeSelector := map[string]string{}
	jobAccelerator := instanceType["accelerator"]["model"]
	jobGPUType := instanceType["accelerator"]["type"]
	jobGPUCount := instanceType["accelerator"]["count"]
	jobLimits := kubernetes_core.ResourceList{}
	jobLimits[kubernetes_core.ResourceName("memory")] = kubernetes_resource.MustParse(instanceType["memory"]["amount"])
	jobLimits[kubernetes_core.ResourceName("cpu")] = kubernetes_resource.MustParse(instanceType["cores"]["count"])
	if diskAmount := d.Get("instance_hdd_size").(int); diskAmount > 0 {
		jobLimits[kubernetes_core.ResourceName("ephemeral-storage")] = kubernetes_resource.MustParse(string(diskAmount) + "G")
	}
	jobImageName := d.Get("name").(string)
	jobName := d.Get("name").(string)
	jobImage := d.Get("image").(string)
	if jobImage == "" {
		jobImage = "dvcorg/cml-py3"
	}

	jobStartupScript := d.Get("startup_script").(string)

	jobReadinessProbe := &kubernetes_core.Probe{
		FailureThreshold: 5,
		SuccessThreshold: 1,
		TimeoutSeconds:   1,
		PeriodSeconds:    10,
		Handler: kubernetes_core.Handler{
			Exec: &kubernetes_core.ExecAction{
				Command: []string{
					"sh", "-c", jobReadinessCommand,
				},
			},
		},
	}


	if jobGPUCount != "0" {
		jobLimits[kubernetes_core.ResourceName(jobGPUType)] = kubernetes_resource.MustParse(jobGPUCount)
		if jobAccelerator != "" {
			jobNodeSelector = map[string]string{"accelerator": jobAccelerator}
		}
	}

	jobTTLSecondsAfterFinished := int32(0)
	jobBackoffLimit := int32(1)
	jobCompletions := int32(1)
	jobParallelism := int32(1)
	jobGracePeriod := int64(30)

	job := kubernetes_batch.Job{
		ObjectMeta: kubernetes_meta.ObjectMeta{
			Name:      jobName,
			Namespace: jobNamespace,
		},
		Spec: kubernetes_batch.JobSpec{
			BackoffLimit: &jobBackoffLimit,
			Completions: &jobCompletions,
			Parallelism: &jobParallelism,
			TTLSecondsAfterFinished: &jobTTLSecondsAfterFinished,
			Template: kubernetes_core.PodTemplateSpec{
				Spec: kubernetes_core.PodSpec{
					TerminationGracePeriodSeconds: &jobGracePeriod,
					RestartPolicy:                 "Never",
					NodeSelector:                  jobNodeSelector,
					Containers: []kubernetes_core.Container{
						{
							Name:  jobImageName,
							Image: jobImage,
							Resources: kubernetes_core.ResourceRequirements{
								Limits: jobLimits,
								Requests: kubernetes_core.ResourceList{
									kubernetes_core.ResourceName("memory"): kubernetes_resource.MustParse("0"),
									kubernetes_core.ResourceName("cpu"): kubernetes_resource.MustParse("0"),
								},
							},
							ReadinessProbe: jobReadinessProbe,
							Env: []kubernetes_core.EnvVar{
								kubernetes_core.EnvVar{
									Name:  "RUNNER_COMMAND",
									Value: jobStartupScript,
								},
							},
							Command: []string{
								"bash", "-c", "base64 -d <<< \"$RUNNER_COMMAND\" | bash",
							},
						},
					},
				},
			},
		},
	}

	log.Printf("[INFO] Creating new Job: %#v", job)

	out, err := conn.BatchV1().Jobs(jobNamespace).Create(ctx, &job, kubernetes_meta.CreateOptions{})
	if err != nil {
		return fmt.Errorf("Failed to create Job! API error: %s", err)
	}
	log.Printf("[INFO] Submitted new job: %#v", out)

	d.SetId(buildId(out.ObjectMeta))
	controllerUID := out.GetObjectMeta().GetLabels()["controller-uid"]
	waitSelector := fmt.Sprintf("controller-uid=%s", controllerUID)

	err = WaitForPods(conn, 1*time.Second, d.Timeout(terraform_schema.TimeoutCreate), jobNamespace, waitSelector)
	if err != nil {
		return err
	}

	return ResourceMachineRead(ctx, d, meta)
}

func ResourceMachineDelete(ctx context.Context, d *terraform_schema.ResourceData, meta interface{}) error {
	conn, err := kubernetesClient()
	if err != nil {
		return err
	}
	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	propagationPolicy := kubernetes_meta.DeletePropagationForeground

	log.Printf("[INFO] Deleting job: %#v", name)
	err = conn.BatchV1().Jobs(namespace).Delete(ctx, name, kubernetes_meta.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil {
		return fmt.Errorf("Failed to delete Job! API error: %s", err)
	}

	err = terraform_resource.RetryContext(ctx, d.Timeout(terraform_schema.TimeoutDelete), func() *terraform_resource.RetryError {
		_, err := conn.BatchV1().Jobs(namespace).Get(ctx, name, kubernetes_meta.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
				return nil
			}
			return terraform_resource.NonRetryableError(err)
		}

		e := fmt.Errorf("Job %s still exists", name)
		return terraform_resource.RetryableError(e)
	})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Job %s deleted", name)

	d.SetId("")
	return nil
}

func ResourceMachineRead(ctx context.Context, d *terraform_schema.ResourceData, meta interface{}) error {
	exists, err := ResourceMachineExists(ctx, d, meta)
	if err != nil {
		return err
	}
	if !exists {
		d.SetId("")
		return nil
	}
	conn, err := kubernetesClient()
	if err != nil {
		return err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading job %s", name)
	job, err := conn.BatchV1().Jobs(namespace).Get(ctx, name, kubernetes_meta.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return fmt.Errorf("Failed to read Job! API error: %s", err)
	}
	log.Printf("[INFO] Received job: %#v", job)

	// Remove server-generated labels unless using manual selector
	if _, ok := d.GetOk("spec.0.manual_selector"); !ok {
		labels := job.ObjectMeta.Labels

		if _, ok := labels["controller-uid"]; ok {
			delete(labels, "controller-uid")
		}

		if _, ok := labels["job-name"]; ok {
			delete(labels, "job-name")
		}

		labels = job.Spec.Selector.MatchLabels

		if _, ok := labels["controller-uid"]; ok {
			delete(labels, "controller-uid")
		}
	}

	if err != nil {
		return err
	}
	return nil
}

func ResourceMachineExists(ctx context.Context, d *terraform_schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := kubernetesClient()
	if err != nil {
		return false, err
	}
	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}
	log.Printf("[INFO] Checking job %s", name)
	_, err = conn.BatchV1().Jobs(namespace).Get(ctx, name, kubernetes_meta.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func kubernetesClient() (kubernetes.Interface, error) {
	configurationBytes := []byte(os.Getenv("KUBERNETES_CONFIGURATION"))
	configuration, err := kubernetes_clientcmd.RESTConfigFromKubeConfig(configurationBytes)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(configuration)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func WaitForPods(client kubernetes.Interface, interval time.Duration, timeout time.Duration, namespace string, selector string,) error {
	pods, err := client.
		CoreV1().Pods(namespace).
		List(context.TODO(), kubernetes_meta.ListOptions{LabelSelector: selector})

	if err != nil {
		return err
	} else if len(pods.Items) == 0 {
		return fmt.Errorf("no pods in %s matching %s", namespace, selector)
	}

	for _, item := range pods.Items {

		function := func() (bool, error) {
			pod, err := client.
				CoreV1().
				Pods(namespace).
				Get(context.TODO(), item.Name, kubernetes_meta.GetOptions{})

			if err != nil {
				return false, err
			}
			fmt.Printf("%s\n", selector)

			for _, condition := range pod.Status.Conditions {
				if condition.Type == "Ready" {
					return condition.Status == "True", nil
				}
			}
			return false, nil
		}

		if err := kubernetes_wait.PollImmediate(interval, timeout, function); err != nil {
			return err
		}
	}
	return nil
}

func getInstanceType(instanceType string, instanceGPU string) (map[string]map[string]string, error) {
	instanceTypes := make(map[string]map[string]map[string]string)

	instanceTypes["m"] = map[string]map[string]string{
		"accelerator": map[string]string{
			"count": "0",
			"type":  "",
			"model": "",
		},
		"cores":  map[string]string{"count": "8"},
		"memory": map[string]string{"amount": "32Gi"},
	}
	instanceTypes["l"] = map[string]map[string]string{
		"accelerator": map[string]string{
			"count": "0",
			"type":  "",
			"model": "",
		},
		"cores":  map[string]string{"count": "32"},
		"memory": map[string]string{"amount": "128Gi"},
	}
	instanceTypes["xl"] = map[string]map[string]string{
		"accelerator": map[string]string{
			"count": "0",
			"type":  "",
			"model": "",
		},
		"cores":  map[string]string{"count": "64"},
		"memory": map[string]string{"amount": "256Gi"},
	}
	instanceTypes["mk80"] = map[string]map[string]string{
		"accelerator": map[string]string{
			"count": "1",
			"type":  "nvidia.com/gpu",
			"model": "nvidia-tesla-k80",
		},
		"cores":  map[string]string{"count": "4"},
		"memory": map[string]string{"amount": "64Gi"},
	}
	instanceTypes["lk80"] = map[string]map[string]string{
		"accelerator": map[string]string{
			"count": "8",
			"type":  "nvidia.com/gpu",
			"model": "nvidia-tesla-k80",
		},
		"cores":  map[string]string{"count": "32"},
		"memory": map[string]string{"amount": "512Gi"},
	}
	instanceTypes["xlk80"] = map[string]map[string]string{
		"accelerator": map[string]string{
			"count": "16",
			"type":  "nvidia.com/gpu",
			"model": "nvidia-tesla-k80",
		},
		"cores":  map[string]string{"count": "64"},
		"memory": map[string]string{"amount": "768Gi"},
	}
	instanceTypes["mtesla"] = map[string]map[string]string{
		"accelerator": map[string]string{
			"count": "1",
			"type":  "nvidia.com/gpu",
			"model": "nvidia-tesla-v100",
		},
		"cores":  map[string]string{"count": "8"},
		"memory": map[string]string{"amount": "64Gi"},
	}
	instanceTypes["ltesla"] = map[string]map[string]string{
		"accelerator": map[string]string{
			"count": "4",
			"type":  "nvidia.com/gpu",
			"model": "nvidia-tesla-v100",
		},
		"cores":  map[string]string{"count": "32"},
		"memory": map[string]string{"amount": "256Gi"},
	}
	instanceTypes["xltesla"] = map[string]map[string]string{
		"accelerator": map[string]string{
			"count": "8",
			"type":  "nvidia.com/gpu",
			"model": "nvidia-tesla-v100",
		},
		"cores":  map[string]string{"count": "64"},
		"memory": map[string]string{"amount": "512Gi"},
	}

	if val, ok := instanceTypes[instanceType+instanceGPU]; ok {
		return val, nil
	}

	return nil, fmt.Errorf("invalid instance type")
}

func idParts(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		err := fmt.Errorf("Unexpected ID format (%q), expected %q.", id, "namespace/name")
		return "", "", err
	}

	return parts[0], parts[1], nil
}

func buildId(meta kubernetes_meta.ObjectMeta) string {
	return meta.Namespace + "/" + meta.Name
}
