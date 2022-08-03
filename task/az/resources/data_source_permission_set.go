package resources

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-30/compute"

	"terraform-provider-iterative/task/az/client"
)

// validateARMID is a regular expression for validating user-assigned identity ids.
var validateARMID = regexp.MustCompile("/subscriptions/.*/resourceGroups/.*/providers/Microsoft.ManagedIdentity/userAssignedIdentities/.*")

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
	identities := strings.Split(ps.Identifier, ",")
	identityMap := map[string]*compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{}
	for _, identity := range identities {
		identity = strings.TrimSpace(identity)
		if identity == "" {
			continue
		}
		if !validateARMID.MatchString(identity) {
			return fmt.Errorf("invalid user-assigned identity id: %q", identity)
		}
		identityMap[identity] = &compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{}
	}
	if len(identityMap) == 0 {
		ps.Resource = nil
		return nil
	}
	ps.Resource = &compute.VirtualMachineScaleSetIdentity{
		UserAssignedIdentities: identityMap,
	}
	return nil
}
