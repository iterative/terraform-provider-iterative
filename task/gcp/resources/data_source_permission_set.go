package resources

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-iterative/task/gcp/client"

	"google.golang.org/api/compute/v1"
)

var (
	scopeMap = map[string]string{
		"bigquery":              "https://www.googleapis.com/auth/bigquery",
		"cloud-platform":        "https://www.googleapis.com/auth/cloud-platform",
		"cloud-source-repos":    "https://www.googleapis.com/auth/source.full_control",
		"cloud-source-repos-ro": "https://www.googleapis.com/auth/source.read_only",
		"compute-ro":            "https://www.googleapis.com/auth/compute.readonly",
		"compute-rw":            "https://www.googleapis.com/auth/compute",
		"datastore":             "https://www.googleapis.com/auth/datastore",
		"logging-write":         "https://www.googleapis.com/auth/logging.write",
		"monitoring":            "https://www.googleapis.com/auth/monitoring",
		"monitoring-read":       "https://www.googleapis.com/auth/monitoring.read",
		"monitoring-write":      "https://www.googleapis.com/auth/monitoring.write",
		"pubsub":                "https://www.googleapis.com/auth/pubsub",
		"service-control":       "https://www.googleapis.com/auth/servicecontrol",
		"service-management":    "https://www.googleapis.com/auth/service.management.readonly",
		"sql":                   "https://www.googleapis.com/auth/sqlservice",
		"sql-admin":             "https://www.googleapis.com/auth/sqlservice.admin",
		"storage-full":          "https://www.googleapis.com/auth/devstorage.full_control",
		"storage-ro":            "https://www.googleapis.com/auth/devstorage.read_only",
		"storage-rw":            "https://www.googleapis.com/auth/devstorage.read_write",
		"taskqueue":             "https://www.googleapis.com/auth/taskqueue",
		"trace":                 "https://www.googleapis.com/auth/trace.append",
		"useraccounts-ro":       "https://www.googleapis.com/auth/cloud.useraccounts.readonly",
		"useraccounts-rw":       "https://www.googleapis.com/auth/cloud.useraccounts",
		"userinfo-email":        "https://www.googleapis.com/auth/userinfo.email",
	}
)

func parseScopes(scopes []string) []string {
	fullScopes := make([]string, len(scopes))
	for i, scope := range scopes {
		fullScopes[i] = func(s string) string {
			if match, ok := scopeMap[s]; ok {
				return match
			}
			return s
		}(scope)
	}
	return fullScopes
}

func NewPermissionSet(client *client.Client, identifier string) *PermissionSet {
	return &PermissionSet{
		client:     client,
		Identifier: identifier,
	}
}

type PermissionSet struct {
	client     *client.Client
	Identifier string
	Resource   []*compute.ServiceAccount
}

func (ps *PermissionSet) Read(ctx context.Context) error {
	permissionSet := ps.Identifier
	if permissionSet == "" {
		ps.Resource = nil
		return nil
	}
	sStr := strings.Split(permissionSet, ",")
	if len(sStr) == 1 {
		return fmt.Errorf("at least one scope is required")
	}
	sStr[1] = strings.Split(sStr[1], "=")[1]
	ps.Resource = []*compute.ServiceAccount{
		{
			Email:  sStr[0],
			Scopes: parseScopes(sStr[1:]),
		},
	}
	return nil
}
