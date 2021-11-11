package task

import (
	"context"
	"fmt"
	"net"

	"terraform-provider-iterative/task/aws"
	"terraform-provider-iterative/task/az"
	"terraform-provider-iterative/task/gcp"
	"terraform-provider-iterative/task/k8s"

	"terraform-provider-iterative/task/universal"
	"terraform-provider-iterative/task/universal/ssh"
)

func NewTask(ctx context.Context, cloud universal.Cloud, identifier string, task universal.Task) (Task, error) {
	switch cloud.Provider {
	case universal.ProviderAWS:
		return aws.NewTask(ctx, cloud, identifier, task)
	case universal.ProviderAZ:
		return az.NewTask(ctx, cloud, identifier, task)
	case universal.ProviderGCP:
		return gcp.NewTask(ctx, cloud, identifier, task)
	case universal.ProviderK8S:
		return k8s.NewTask(ctx, cloud, identifier, task)
	default:
		return nil, fmt.Errorf("unknown provider: %#v", cloud.Provider)
	}
}

type Task interface {
	Read(ctx context.Context) error

	Create(ctx context.Context) error
	Delete(ctx context.Context) error

	Push(ctx context.Context, source string, unsafe bool) error
	Pull(ctx context.Context, destination string) error

	Logs(ctx context.Context) ([]string, error)

	// Not useful for Kubernetes.
	Stop(ctx context.Context) error

	// To be refactored.
	GetAddresses(ctx context.Context) []net.IP
	GetEvents(ctx context.Context) []universal.Event
	GetStatus(ctx context.Context) map[string]int
	GetKeyPair(ctx context.Context) (*ssh.DeterministicSSHKeyPair, error)
	GetIdentifier(ctx context.Context) string
}
