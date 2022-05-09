package resources

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-30/compute"

	"terraform-provider-iterative/task/az/client"
)

func NewPermissionSet(client *client.Client, identifer string) *PermissionSet {
	ps := new(PermissionSet)
	ps.Client = client
	ps.Identifier = identifer
	return ps
}

type PermissionSet struct {
	Client     *client.Client
	Identifier string
	Resource   *compute.VirtualMachineScaleSetIdentity
}

func (ps *PermissionSet) Read(ctx context.Context) error {
	if ps.Identifier != "" {
		return fmt.Errorf("not yet implemented")
	}
	ps.Resource = nil
	return nil
}
