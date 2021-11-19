package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/common"
)

func NewDefaultVPCSubnet(client *client.Client, defaultVpc *DefaultVPC) *DefaultVPCSubnet {
	d := new(DefaultVPCSubnet)
	d.Client = client
	d.Dependencies.DefaultVPC = defaultVpc
	return d
}

type DefaultVPCSubnet struct {
	Client       *client.Client
	Resource     *types.Subnet
	Dependencies struct {
		*DefaultVPC
	}
}

func (d *DefaultVPCSubnet) Read(ctx context.Context) error {
	input := ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{aws.ToString(d.Dependencies.DefaultVPC.Resource.VpcId)},
			},
		},
	}

	subnets, err := d.Client.Services.EC2.DescribeSubnets(ctx, &input)
	if err != nil {
		return err
	}

	if len(subnets.Subnets) < 1 {
		return common.NotFoundError
	}

	for _, subnet := range subnets.Subnets {
		if aws.ToInt32(subnet.AvailableIpAddressCount) > 0 && aws.ToBool(subnet.MapPublicIpOnLaunch) {
			d.Resource = &subnet
			return nil
		}
	}

	return common.NotFoundError
}
