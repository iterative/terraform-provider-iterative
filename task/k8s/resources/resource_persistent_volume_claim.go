package resources

import (
	"context"
	"strconv"

	kubernetes_core "k8s.io/api/core/v1"
	kubernetes_errors "k8s.io/apimachinery/pkg/api/errors"
	kubernetes_resource "k8s.io/apimachinery/pkg/api/resource"
	kubernetes_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/k8s/client"
)

func NewPersistentVolumeClaim(client *client.Client, identifier common.Identifier, storageClass string, size uint64, many bool) *PersistentVolumeClaim {
	p := new(PersistentVolumeClaim)
	p.Client = client
	p.Identifier = identifier.Long()
	p.Attributes.StorageClass = storageClass
	p.Attributes.Size = size
	p.Attributes.Many = many
	return p
}

type PersistentVolumeClaim struct {
	Client     *client.Client
	Identifier string
	Attributes struct {
		StorageClass string
		Size         uint64
		Many         bool
	}
	Dependencies struct{}
	Resource     *kubernetes_core.PersistentVolumeClaim
}

func (p *PersistentVolumeClaim) Create(ctx context.Context) error {
	accessMode := kubernetes_core.ReadWriteOnce
	if p.Attributes.Many {
		accessMode = kubernetes_core.ReadWriteMany
	}

	persistentVolumeClaimInput := kubernetes_core.PersistentVolumeClaim{
		ObjectMeta: kubernetes_meta.ObjectMeta{
			Name:        p.Identifier,
			Namespace:   p.Client.Namespace,
			Labels:      p.Client.Tags,
			Annotations: p.Client.Tags,
		},
		Spec: kubernetes_core.PersistentVolumeClaimSpec{
			AccessModes: []kubernetes_core.PersistentVolumeAccessMode{accessMode},
			Resources: kubernetes_core.ResourceRequirements{
				Requests: kubernetes_core.ResourceList{
					kubernetes_core.ResourceStorage: kubernetes_resource.MustParse(strconv.Itoa(int(p.Attributes.Size)) + "G"),
				},
			},
		},
	}

	if p.Attributes.StorageClass != "" {
		persistentVolumeClaimInput.Spec.StorageClassName = &p.Attributes.StorageClass
	}

	_, err := p.Client.Services.Core.PersistentVolumeClaims(p.Client.Namespace).Create(ctx, &persistentVolumeClaimInput, kubernetes_meta.CreateOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 409 {
			return nil
		}
		return err
	}

	return p.Read(ctx)
}

func (p *PersistentVolumeClaim) Read(ctx context.Context) error {
	persistentVolumeClaim, err := p.Client.Services.Core.PersistentVolumeClaims(p.Client.Namespace).Get(ctx, p.Identifier, kubernetes_meta.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return common.NotFoundError
		}
		return err
	}

	p.Resource = persistentVolumeClaim
	return nil
}

func (p *PersistentVolumeClaim) Delete(ctx context.Context) error {
	err := p.Client.Services.Core.PersistentVolumeClaims(p.Client.Namespace).Delete(ctx, p.Identifier, kubernetes_meta.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return nil
		}
		return err
	}
	return nil
}
