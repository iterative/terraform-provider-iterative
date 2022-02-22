package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-06-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
)

func NewResourceGroup(client *client.Client, identifier common.Identifier) *ResourceGroup {
	r := new(ResourceGroup)
	r.Client = client
	r.Identifier = identifier.Long()
	return r
}

type ResourceGroup struct {
	Client     *client.Client
	Identifier string
	Resource   *resources.Group
}

func (r *ResourceGroup) Create(ctx context.Context) error {
	resourceGroup, err := r.Client.Services.Groups.CreateOrUpdate(
		ctx,
		r.Identifier,
		resources.Group{
			Location: to.StringPtr(r.Client.Region),
			Tags:     r.Client.Tags,
		})
	if err != nil {
		return err
	}

	r.Resource = &resourceGroup
	return nil
}

func (r *ResourceGroup) Read(ctx context.Context) error {
	resourceGroup, err := r.Client.Services.Groups.Get(ctx, r.Identifier)
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
	resourceGroupDeleteFuture, err := r.Client.Services.Groups.Delete(ctx, r.Identifier, "")
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return nil
		}
		return err
	}

	err = resourceGroupDeleteFuture.WaitForCompletionRef(ctx, r.Client.Services.Groups.Client)
	r.Resource = nil
	return err
}
