package client

import (
	"context"
	"os"

	"k8s.io/client-go/kubernetes"
	batchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"terraform-provider-iterative/task/universal"
)

func New(ctx context.Context, cloud universal.Cloud, tags map[string]string) (*Client, error) {
	config, err := clientcmd.NewClientConfigFromBytes(
		[]byte(os.Getenv("KUBERNETES_CONFIGURATION")),
	)
	if err != nil {
		config = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{},
		)
	}

	clientConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	namespace, _, err := config.Namespace()
	if err != nil {
		return nil, err
	}

	c := new(Client)
	c.Cloud = cloud
	c.Namespace = namespace
	c.Tags = tags
	c.Config = &config
	c.ClientConfig = clientConfig
	c.ClientSet = client
	c.Services.Batch = client.BatchV1()
	c.Services.Core = client.CoreV1()
	return c, nil
}

type Client struct {
	Cloud        universal.Cloud
	Namespace    string
	Tags         map[string]string
	Config       *clientcmd.ClientConfig
	ClientConfig *rest.Config
	ClientSet    *kubernetes.Clientset
	Services     struct {
		Batch batchv1.BatchV1Interface
		Core  corev1.CoreV1Interface
	}
}
