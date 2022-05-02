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
	// https://pkg.go.dev/github.com/Azure/azure-sdk-for-go@v58.1.0+incompatible/services/compute/mgmt/2020-06-30/compute#VirtualMachineScaleSetIdentity
	// https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview
	// https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/how-manage-user-assigned-managed-identities?pivots=identity-mi-methods-azp
	if ps.Identifier != "" {
		return fmt.Errorf("not yet implemented")
	}
	ps.Resource = nil
	return nil
}
