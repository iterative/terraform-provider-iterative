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

func ListConfigMaps(ctx context.Context, client *client.Client) ([]common.Identifier, error) {
	cmaps, err := client.Services.Core.ConfigMaps(client.Namespace).List(ctx, kubernetes_meta.ListOptions{})
	if err != nil {
		return nil, err
	}

	ids := []common.Identifier{}
	for _, cmap := range cmaps.Items {
		if id, err := common.ParseIdentifier(cmap.ObjectMeta.Name); err == nil {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

func NewConfigMap(client *client.Client, identifier common.Identifier, data map[string]string) *ConfigMap {
	return &ConfigMap{
		client:     client,
		Identifier: identifier.Long(),
		Attributes: data,
	}
}

type ConfigMap struct {
	client     *client.Client
	Identifier string
	Attributes map[string]string
	Resource   *kubernetes_core.ConfigMap
}

func (c *ConfigMap) Create(ctx context.Context) error {
	configMapInput := kubernetes_core.ConfigMap{
		ObjectMeta: kubernetes_meta.ObjectMeta{
			Name:        c.Identifier,
			Namespace:   c.client.Namespace,
			Labels:      c.client.Tags,
			Annotations: c.client.Tags,
		},
		Data: c.Attributes,
	}

	_, err := c.client.Services.Core.ConfigMaps(c.client.Namespace).Create(ctx, &configMapInput, kubernetes_meta.CreateOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 409 {
			return nil
		}
		return err
	}

	return c.Read(ctx)
}

func (c *ConfigMap) Read(ctx context.Context) error {
	configMap, err := c.client.Services.Core.ConfigMaps(c.client.Namespace).Get(ctx, c.Identifier, kubernetes_meta.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return common.NotFoundError
		}
		return err
	}

	c.Resource = configMap
	return nil
}

func (c *ConfigMap) Delete(ctx context.Context) error {
	err := c.client.Services.Core.ConfigMaps(c.client.Namespace).Delete(ctx, c.Identifier, kubernetes_meta.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return nil
		}
		return err
	}
	return nil
}
