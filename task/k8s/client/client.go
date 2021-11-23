package client

import (
	"context"
	"os"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	batchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"terraform-provider-iterative/task/common"
)

func New(ctx context.Context, cloud common.Cloud, tags map[string]string) (*Client, error) {
	kubeconfig := os.Getenv("KUBERNETES_CONFIGURATION") // Legacy; deprecated.
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG_DATA")
	}

	config, err := clientcmd.NewClientConfigFromBytes([]byte(kubeconfig))
	if err != nil || kubeconfig == "" {
		config = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{},
		)
	}

	clientConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}

	clientConfig.APIPath = "/api"
	clientConfig.GroupVersion = &schema.GroupVersion{Version: "v1"}
	clientConfig.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}

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
	Cloud        common.Cloud
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
