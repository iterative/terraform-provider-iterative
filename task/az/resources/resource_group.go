package resources

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
)

func ListResourceGroups(ctx context.Context, client *client.Client) ([]common.Identifier, error) {
	ids := []common.Identifier{}

	for pager := client.Services.ResourceGroups.NewListPager(nil); pager.More(); {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, group := range page.ResourceGroupListResult.Value {
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
	Resource   *armresources.ResourceGroup
}

func (r *ResourceGroup) Create(ctx context.Context) error {
	response, err := r.client.Services.ResourceGroups.CreateOrUpdate(
		ctx,
		r.Identifier,
		armresources.ResourceGroup{
			Location: to.Ptr(r.client.Region),
			Tags:     r.client.Tags,
		},
		nil,
	)
	if err != nil {
		return err
	}

	r.Resource = &response.ResourceGroup
	return nil
}

func (r *ResourceGroup) Read(ctx context.Context) error {
	response, err := r.client.Services.ResourceGroups.Get(ctx, r.Identifier, nil)
	if err != nil {
		var e *azcore.ResponseError
		if errors.As(err, &e) && e.RawResponse.StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	r.Resource = &response.ResourceGroup
	return nil
}

func (r *ResourceGroup) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (r *ResourceGroup) Delete(ctx context.Context) error {
	poller, err := r.client.Services.ResourceGroups.BeginDelete(ctx, r.Identifier, nil)
	if err != nil {
		var e *azcore.ResponseError
		if errors.As(err, &e) && e.RawResponse.StatusCode == 404 {
			return nil
		}
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	r.Resource = nil
	return err
}
