package resources

import (
	"context"
	"fmt"
	"net/http"

	kubernetes_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/k8s/client"
)

// NewPermissionSet creates a new permission set.
func NewPermissionSet(client *client.Client, identifier string) *PermissionSet {
	ps := new(PermissionSet)
	ps.client = client
	ps.Identifier = identifier
	return ps
}

// PermissionSet matches the provided service account name to an existing service account.
type PermissionSet struct {
	client     *client.Client
	Identifier string
	Resource   struct {
		ServiceAccountName           string
		AutomountServiceAccountToken *bool
	}
}

// Read verifies the service account.
func (ps *PermissionSet) Read(ctx context.Context) error {
	if ps.Identifier == "" {
		return nil
	}
	account, err := ps.client.Services.Core.ServiceAccounts(ps.client.Namespace).Get(ctx, ps.Identifier, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*kubernetes_errors.StatusError); ok && statusErr.ErrStatus.Code == http.StatusNotFound {
			return fmt.Errorf("service account %q does not exist in namespace %q: %w",
				ps.Identifier, ps.client.Namespace, common.NotFoundError)
		}
		return fmt.Errorf("failed to lookup service account %q in namespace %q: %w",
			ps.Identifier, ps.client.Namespace, common.NotFoundError)

	}
	ps.Resource.ServiceAccountName = ps.Identifier
	ps.Resource.AutomountServiceAccountToken = account.AutomountServiceAccountToken
	return nil
}
