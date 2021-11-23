package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/common"
)

func NewDefaultVPC(client *client.Client) *DefaultVPC {
	d := new(DefaultVPC)
	d.Client = client
	return d
}

type DefaultVPC struct {
	Client   *client.Client
	Resource *types.Vpc
}

func (d *DefaultVPC) Read(ctx context.Context) error {
	input := ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("is-default"),
				Values: []string{"true"},
			},
		},
	}

	vpcs, err := d.Client.Services.EC2.DescribeVpcs(ctx, &input)
	if err != nil {
		return err
	}

	if len(vpcs.Vpcs) < 1 {
		return common.NotFoundError
	}

	d.Resource = &vpcs.Vpcs[0]
	return nil
}
