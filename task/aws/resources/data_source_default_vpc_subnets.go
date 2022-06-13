package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/common"
)

func NewDefaultVPCSubnets(client *client.Client, defaultVpc *DefaultVPC) *DefaultVPCSubnets {
	d := new(DefaultVPCSubnets)
	d.Client = client
	d.Dependencies.DefaultVPC = defaultVpc
	return d
}

type DefaultVPCSubnets struct {
	Client       *client.Client
	Resource     []*types.Subnet
	Dependencies struct {
		*DefaultVPC
	}
}

func (d *DefaultVPCSubnets) Read(ctx context.Context) error {
	input := ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{aws.ToString(d.Dependencies.DefaultVPC.Resource.VpcId)},
			},
			{
				Name:   aws.String("default-for-az"),
				Values: []string{"true"},
			},
		},
	}

	subnets, err := d.Client.Services.EC2.DescribeSubnets(ctx, &input)
	if err != nil {
		return err
	}

	d.Resource = nil
	for _, subnet := range subnets.Subnets {
		s := subnet
		if aws.ToInt32(s.AvailableIpAddressCount) > 0 && aws.ToBool(s.MapPublicIpOnLaunch) {
			d.Resource = append(d.Resource, &s)
		}
	}

	if len(d.Resource) < 1 {
		return common.NotFoundError
	}

	return nil
}
