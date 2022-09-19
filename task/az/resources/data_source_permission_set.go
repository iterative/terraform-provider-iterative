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
var validateARMID = regexp.MustCompile(`^/subscriptions/(\w{8}-\w{4}-\w{4}-\w{4}-\w{12})/resourceGroups/(.*)/providers/Microsoft.ManagedIdentity/userAssignedIdentities/(.*)`)

// ValidateARMID validates the user-assigned identity value.
func ValidateARMID(id string) error {
	if !validateARMID.MatchString(id) {
		return fmt.Errorf("invalid user-assigned identity id: %q", id)
	}
	return nil
}

func NewPermissionSet(client *client.Client, identifer string) *PermissionSet {
	return &PermissionSet{
		client:     client,
		Identifier: identifer,
	}
}

type PermissionSet struct {
	client     *client.Client
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
		if err := ValidateARMID(identity); err != nil {
			return err
		}

		identityMap[identity] = &compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{}
	}
	if len(identityMap) == 0 {
		ps.Resource = nil
		return nil
	}
	ps.Resource = &compute.VirtualMachineScaleSetIdentity{
		UserAssignedIdentities: identityMap,
		Type:                   compute.ResourceIdentityTypeSystemAssignedUserAssigned,
	}
	return nil
}
