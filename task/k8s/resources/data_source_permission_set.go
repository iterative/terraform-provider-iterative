package resources

import (
	"context"
	"fmt"
	"terraform-provider-iterative/task/k8s/client"
)

func NewPermissionSet(client *client.Client, identifier string) *PermissionSet {
	return &PermissionSet{
		client:     client,
		Identifier: identifier,
	}
}

type PermissionSet struct {
	client     *client.Client
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
