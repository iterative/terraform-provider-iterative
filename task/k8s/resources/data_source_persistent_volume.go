package resources

import (
	"context"

	kubernetes_core "k8s.io/api/core/v1"
	kubernetes_errors "k8s.io/apimachinery/pkg/api/errors"
	kubernetes_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/iterative/terraform-provider-iterative/task/common"
	"github.com/iterative/terraform-provider-iterative/task/k8s/client"
)

// NewExistingPersistentVolumeClaim creates a new ExistingPersistentVolumeClaim object.
func NewExistingPersistentVolumeClaim(client *client.Client, storageParams common.RemoteStorage) *ExistingPersistentVolumeClaim {
	return &ExistingPersistentVolumeClaim{
		client: client,
		params: storageParams,
	}
}

// ExistingPersistentVolumeClaim refers to a pre-allocated persistent volume to be used
// as storage for the job.
type ExistingPersistentVolumeClaim struct {
	client   *client.Client
	params   common.RemoteStorage
	resource *kubernetes_core.PersistentVolumeClaim
}

// Read verifies the persistent volume.
func (p *ExistingPersistentVolumeClaim) Read(ctx context.Context) error {
	persistentVolumeClaim, err := p.client.Services.Core.PersistentVolumeClaims(p.client.Namespace).Get(ctx, p.params.Container, kubernetes_meta.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return common.NotFoundError
		}
		return err
	}

	p.resource = persistentVolumeClaim
	return nil
}

// VolumeInfo returns information for attaching the persistent volume claim to the job.
func (p *ExistingPersistentVolumeClaim) VolumeInfo(ctx context.Context) (string /*subpath*/, *kubernetes_core.PersistentVolumeClaimVolumeSource) {
	pvc := &kubernetes_core.PersistentVolumeClaimVolumeSource{
		ClaimName: p.params.Container,
	}
	return p.params.Path, pvc

}
