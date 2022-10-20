package task

import (
	"context"
	"fmt"
	"net"

	"terraform-provider-iterative/task/aws"
	"terraform-provider-iterative/task/az"
	"terraform-provider-iterative/task/gcp"
	"terraform-provider-iterative/task/k8s"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/ssh"
)

func List(ctx context.Context, cloud common.Cloud) ([]common.Identifier, error) {
	switch cloud.Provider {
	case common.ProviderAWS:
		return aws.List(ctx, cloud)
	case common.ProviderAZ:
		return az.List(ctx, cloud)
	case common.ProviderGCP:
		return gcp.List(ctx, cloud)
	case common.ProviderK8S:
		return k8s.List(ctx, cloud)
	default:
		return nil, fmt.Errorf("unknown provider: %#v", cloud.Provider)
	}
}

func New(ctx context.Context, cloud common.Cloud, identifier common.Identifier, task common.Task) (Task, error) {
	switch cloud.Provider {
	case common.ProviderAWS:
		return aws.New(ctx, cloud, identifier, task)
	case common.ProviderAZ:
		return az.New(ctx, cloud, identifier, task)
	case common.ProviderGCP:
		return gcp.New(ctx, cloud, identifier, task)
	case common.ProviderK8S:
		return k8s.New(ctx, cloud, identifier, task)
	default:
		return nil, fmt.Errorf("unknown provider: %#v", cloud.Provider)
	}
}

type Task interface {
	Read(ctx context.Context) error

	Create(ctx context.Context) error
	Delete(ctx context.Context) error

	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// Push uploads the task's workspace to the remote storage.
	Push(ctx context.Context) error
	// Pull downloads the output directory from remote storage.
	Pull(ctx context.Context) error

	Status(ctx context.Context) (common.Status, error)
	Events(ctx context.Context) []common.Event
	Logs(ctx context.Context) ([]string, error)

	// To be refactored.
	GetIdentifier(ctx context.Context) common.Identifier
	GetAddresses(ctx context.Context) []net.IP
	GetKeyPair(ctx context.Context) (*ssh.DeterministicSSHKeyPair, error)
}
