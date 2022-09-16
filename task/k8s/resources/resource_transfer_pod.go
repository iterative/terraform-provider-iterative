package resources

import (
	"context"
	"fmt"

	k8s_apps "k8s.io/api/apps/v1"
	k8s_core "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/k8s/client"
)

// NewTransferDeployment creates a pod used to transfer data between the persistent storage and local.
func NewTransferDeployment(client *client.Client, identifier common.Identifier, persistentVolumeClaim *PersistentVolumeClaim, permissionSet *PermissionSet, task common.Task) *TransferDeployment {
	p := &TransferDeployment{
		Client:     client,
		Identifier: identifier.Long(),
	}
	p.Dependencies.PersistentVolumeClaim = persistentVolumeClaim
	p.Dependencies.PermissionSet = permissionSet
	p.Attributes.Task = task
	return p
}

// TransferDeployment is a deployment with a single pod used to transfer data between the persistent volume claim and local.
type TransferDeployment struct {
	Client     *client.Client
	Identifier string
	Attributes struct {
		Task common.Task
	}
	Dependencies struct {
		PersistentVolumeClaim *PersistentVolumeClaim
		PermissionSet         *PermissionSet
	}
	Resource *k8s_apps.Deployment
}

// Create creates the transfer pod resource.
func (p *TransferDeployment) Create(ctx context.Context) error {
	if p.Attributes.Task.Environment.Directory == "" {
		return fmt.Errorf("output directory not set")
	}

	volumeMounts := []k8s_core.VolumeMount{{
		Name:      p.Identifier,
		MountPath: "/data",
	}}
	volumes := []k8s_core.Volume{{
		Name: p.Identifier,
		VolumeSource: k8s_core.VolumeSource{
			PersistentVolumeClaim: &k8s_core.PersistentVolumeClaimVolumeSource{
				ClaimName: p.Dependencies.PersistentVolumeClaim.Identifier,
			},
		},
	}}

	replicas := int32(1)
	deployment := &k8s_apps.Deployment{
		ObjectMeta: k8s_meta.ObjectMeta{
			Name:        p.Identifier,
			Namespace:   p.Client.Namespace,
			Labels:      p.Client.Tags,
			Annotations: p.Client.Tags,
		},
		Spec: k8s_apps.DeploymentSpec{
			Replicas: &replicas,
			Selector: &k8s_meta.LabelSelector{
				MatchLabels: map[string]string{
					"app": p.Identifier + "-transfer",
				},
			},
			Template: k8s_core.PodTemplateSpec{
				ObjectMeta: k8s_meta.ObjectMeta{
					Labels: map[string]string{
						"app": p.Identifier + "-transfer",
					},
				},
				Spec: k8s_core.PodSpec{
					RestartPolicy:                k8s_core.RestartPolicyAlways,
					ServiceAccountName:           p.Dependencies.PermissionSet.Resource.ServiceAccountName,
					AutomountServiceAccountToken: p.Dependencies.PermissionSet.Resource.AutomountServiceAccountToken,
					Volumes:                      volumes,
					Containers: []k8s_core.Container{{
						Name:            p.Identifier + "-tx",
						Image:           "busybox",
						Command:         []string{"sleep", "86400"},
						VolumeMounts:    volumeMounts,
						ImagePullPolicy: k8s_core.PullIfNotPresent,
					}},
				},
			},
		},
	}
	// Ask Kubernetes to create the job.
	out, err := p.Client.Services.Apps.Deployments(p.Client.Namespace).Create(ctx, deployment, k8s_meta.CreateOptions{})
	if err != nil {
		return err
	}

	p.Resource = out
	return nil
}

// Read updates the information of the transfer deployment.
func (p *TransferDeployment) Read(ctx context.Context) error {
	deployment, err := p.Client.Services.Apps.Deployments(p.Client.Namespace).Get(ctx, p.Identifier, k8s_meta.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*k8s_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return common.NotFoundError
		}
		return err
	}
	p.Resource = deployment
	return nil
}

// Delete removes the transfer deployment.
func (p *TransferDeployment) Delete(ctx context.Context) error {
	_, err := p.Client.Services.Apps.Deployments(p.Client.Namespace).Get(ctx, p.Identifier, k8s_meta.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*k8s_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return nil
		}
		return err
	}

	propagationPolicy := k8s_meta.DeletePropagationForeground

	err = p.Client.Services.Apps.Deployments(p.Client.Namespace).Delete(ctx, p.Identifier, k8s_meta.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil {
		if statusErr, ok := err.(*k8s_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return nil
		}
		return fmt.Errorf("Failed to delete deployment! API error: %w", err)
	}
	return nil
}
