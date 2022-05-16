package resources

import (
	"context"
	"strings"

	kubernetes_core "k8s.io/api/core/v1"
	kubernetes_errors "k8s.io/apimachinery/pkg/api/errors"
	kubernetes_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/k8s/client"
)

func ListConfigMaps(ctx context.Context, client *client.Client) ([]common.Identifier, error) {

	cmaps, err := client.Services.Core.ConfigMaps(client.Namespace).List(ctx, kubernetes_meta.ListOptions{})
	if err != nil {
		return nil, err
	}

	ids := []common.Identifier{}
	for _, cmap := range cmaps.Items {
		if id := common.Identifier(cmap.ObjectMeta.Name); strings.HasPrefix(string(id), "tpi-") && !strings.HasPrefix(string(id), "tpi-tpi-"){
			ids = append(ids, id)
		}
	}
	
	return ids, nil
}

func NewConfigMap(client *client.Client, identifier common.Identifier, data map[string]string) *ConfigMap {
	c := new(ConfigMap)
	c.Client = client
	c.Identifier = identifier.Long()
	c.Attributes = data
	return c
}

type ConfigMap struct {
	Client       *client.Client
	Identifier   string
	Attributes   map[string]string
	Dependencies struct{}
	Resource     *kubernetes_core.ConfigMap
}

func (c *ConfigMap) Create(ctx context.Context) error {
	configMapInput := kubernetes_core.ConfigMap{
		ObjectMeta: kubernetes_meta.ObjectMeta{
			Name:        c.Identifier,
			Namespace:   c.Client.Namespace,
			Labels:      c.Client.Tags,
			Annotations: c.Client.Tags,
		},
		Data: c.Attributes,
	}

	_, err := c.Client.Services.Core.ConfigMaps(c.Client.Namespace).Create(ctx, &configMapInput, kubernetes_meta.CreateOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 409 {
			return nil
		}
		return err
	}

	return c.Read(ctx)
}

func (c *ConfigMap) Read(ctx context.Context) error {
	configMap, err := c.Client.Services.Core.ConfigMaps(c.Client.Namespace).Get(ctx, c.Identifier, kubernetes_meta.GetOptions{})
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
	err := c.Client.Services.Core.ConfigMaps(c.Client.Namespace).Delete(ctx, c.Identifier, kubernetes_meta.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return nil
		}
		return err
	}
	return nil
}
