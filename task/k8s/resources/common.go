package resources

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"terraform-provider-iterative/task/k8s/client"
)

// Wait for the pods matching the specified selector to pass their respective readiness probes.
func WaitForPods(ctx context.Context, client *client.Client, interval time.Duration, timeout time.Duration, namespace string, selector string) (string, error) {
	// Retrieve all the pods matching the given selector.
	pods, err := client.Services.Core.Pods(namespace).
		List(ctx, v1.ListOptions{LabelSelector: selector})

	if err != nil {
		return "", err
	} else if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods in %s matching %s", namespace, selector)
	}

	var podName string
	// Await the matching pods sequentially and return as soon as one of it fails the readiness check.
	for _, currentPod := range pods.Items {
		// Define a closured function in charge of checking the state of the current pod.
		function := func() (bool, error) {
			pod, err := client.Services.Core.Pods(namespace).
				Get(ctx, currentPod.Name, v1.GetOptions{})

			if err != nil {
				return false, err
			}

			for _, condition := range pod.Status.Conditions {
				if condition.Type == "Ready" {
					podName = currentPod.Name
					return condition.Status == "True", nil
				}
			}
			return false, nil
		}
		// Wait for the pod to be ready by polling its status, and return the error on timeout.
		if err := wait.PollImmediate(interval, timeout, function); err != nil {
			return "", err
		}
	}
	return podName, nil
}
