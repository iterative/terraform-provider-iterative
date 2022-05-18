package resources

import (
	"context"
	"fmt"
	"terraform-provider-iterative/task/k8s/client"
)

func NewPermissionSet(client *client.Client, identifier string) *PermissionSet {
	ps := new(PermissionSet)
	ps.Client = client
	ps.Identifier = identifier
	return ps
}

type PermissionSet struct {
	Client     *client.Client
	Identifier string
	Resource   struct {
		ServiceAccountName           string
		AutomountServiceAccountToken *bool
		flag                         bool
	}
}

func (ps *PermissionSet) Read(ctx context.Context) error {
	ps.Resource.flag = true
	if ps.Identifier == "" {
		ps.Resource.ServiceAccountName = ""
		ps.Resource.AutomountServiceAccountToken = nil
		return nil
	}
	return fmt.Errorf("not yet implemented")
}
