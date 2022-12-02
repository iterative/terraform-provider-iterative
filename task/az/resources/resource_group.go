package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-06-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/iterative/terraform-provider-iterative/task/az/client"
	"github.com/iterative/terraform-provider-iterative/task/common"
)

func ListResourceGroups(ctx context.Context, client *client.Client) ([]common.Identifier, error) {
	ids := []common.Identifier{}

	for page, err := client.Services.Groups.List(ctx, "", nil); page.NotDone(); err = page.Next() {
		if err != nil {
			return nil, err
		}

		for _, group := range page.Values() {
			if id, err := common.ParseIdentifier(*group.Name); err == nil {
				ids = append(ids, id)
			}
		}
	}

	return ids, nil
}

func NewResourceGroup(client *client.Client, identifier common.Identifier) *ResourceGroup {
	return &ResourceGroup{
		client:     client,
		Identifier: identifier.Long(),
	}
}

type ResourceGroup struct {
	client     *client.Client
	Identifier string
	Resource   *resources.Group
}

func (r *ResourceGroup) Create(ctx context.Context) error {
	resourceGroup, err := r.client.Services.Groups.CreateOrUpdate(
		ctx,
		r.Identifier,
		resources.Group{
			Location: to.StringPtr(r.client.Region),
			Tags:     r.client.Tags,
		})
	if err != nil {
		return err
	}

	r.Resource = &resourceGroup
	return nil
}

func (r *ResourceGroup) Read(ctx context.Context) error {
	resourceGroup, err := r.client.Services.Groups.Get(ctx, r.Identifier)
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	r.Resource = &resourceGroup
	return nil
}

func (r *ResourceGroup) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (r *ResourceGroup) Delete(ctx context.Context) error {
	resourceGroupDeleteFuture, err := r.client.Services.Groups.Delete(ctx, r.Identifier, "")
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return nil
		}
		return err
	}

	err = resourceGroupDeleteFuture.WaitForCompletionRef(ctx, r.client.Services.Groups.Client)
	r.Resource = nil
	return err
}
