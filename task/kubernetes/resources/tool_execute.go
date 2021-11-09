package resources

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubectl/pkg/cmd/cp"

	"terraform-provider-iterative/task/kubernetes/client"
)

func CopyFile(client *client.Client, source string, destination string, container string) error {
	client.ClientConfig.APIPath = "/api"
	client.ClientConfig.GroupVersion = &schema.GroupVersion{Version: "v1"}
	client.ClientConfig.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	ioStreams, _, _, _ := genericclioptions.NewTestIOStreams()
	copyOptions := cp.NewCopyOptions(ioStreams)
	copyOptions.Clientset = client.ClientSet
	copyOptions.ClientConfig = client.ClientConfig
	copyOptions.Container = container
	return copyOptions.Run([]string{source, destination})
}
