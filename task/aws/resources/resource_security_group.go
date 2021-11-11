package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/universal"
)

func NewSecurityGroup(client *client.Client, identifier string, defaultVPC *DefaultVPC, firewall universal.Firewall) *SecurityGroup {
	s := new(SecurityGroup)
	s.Client = client
	s.Identifier = universal.NormalizeIdentifier(identifier, true)
	s.Attributes = firewall
	s.Dependencies.DefaultVPC = defaultVPC
	return s
}

type SecurityGroup struct {
	Client       *client.Client
	Identifier   string
	Attributes   universal.Firewall
	Dependencies struct {
		*DefaultVPC
	}
	Resource *types.SecurityGroup
}

func (s *SecurityGroup) Create(ctx context.Context) error {
	if err := s.Read(ctx); err != universal.NotFoundError {
		return err
	}

	createInput := ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(s.Identifier),
		Description: aws.String(s.Identifier),
		VpcId:       s.Dependencies.DefaultVPC.Resource.VpcId,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSecurityGroup,
				Tags:         makeTagSlice(s.Identifier, s.Client.Tags),
			},
		},
	}

	group, err := s.Client.Services.EC2.CreateSecurityGroup(ctx, &createInput)
	if err != nil {
		return err
	}

	describeInput := ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{aws.ToString(group.GroupId)},
	}

	if err := ec2.NewSecurityGroupExistsWaiter(s.Client.Services.EC2).Wait(ctx, &describeInput, s.Client.Cloud.Timeouts.Create); err != nil {
		return err
	}

	if err := s.Read(ctx); err != nil {
		return err
	}

	// Revoke default full egress rule created for every new security group
	revokeEgressInput := ec2.RevokeSecurityGroupEgressInput{
		GroupId:       s.Resource.GroupId,
		IpPermissions: s.generatePermissions(universal.FirewallRule{}),
	}

	if _, err := s.Client.Services.EC2.RevokeSecurityGroupEgress(ctx, &revokeEgressInput); err != nil {
		return err
	}

	egressInput := ec2.AuthorizeSecurityGroupEgressInput{
		GroupId:       s.Resource.GroupId,
		IpPermissions: s.generatePermissions(s.Attributes.Egress),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSecurityGroupRule,
				Tags:         makeTagSlice(s.Identifier, s.Client.Tags),
			},
		},
	}

	if _, err := s.Client.Services.EC2.AuthorizeSecurityGroupEgress(ctx, &egressInput); err != nil {
		return err
	}

	ingressInput := ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:       s.Resource.GroupId,
		IpPermissions: s.generatePermissions(s.Attributes.Ingress),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSecurityGroupRule,
				Tags:         makeTagSlice(s.Identifier, s.Client.Tags),
			},
		},
	}

	if _, err := s.Client.Services.EC2.AuthorizeSecurityGroupIngress(ctx, &ingressInput); err != nil {
		return err
	}

	return nil
}

func (s *SecurityGroup) Read(ctx context.Context) error {
	input := ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{s.Identifier},
			},
		},
	}

	securityGroups, err := s.Client.Services.EC2.DescribeSecurityGroups(ctx, &input)
	if err != nil {
		return err
	}

	if len(securityGroups.SecurityGroups) < 1 {
		return universal.NotFoundError
	}

	s.Resource = &securityGroups.SecurityGroups[0]
	return nil
}

func (s *SecurityGroup) Update(ctx context.Context) error {
	return universal.NotImplementedError
}

func (s *SecurityGroup) Delete(ctx context.Context) error {
	switch err := s.Read(ctx); {
	case err == universal.NotFoundError:
		return nil
	case err != nil:
		return err
	}

	input := ec2.DeleteSecurityGroupInput{
		GroupId: s.Resource.GroupId,
	}

	if _, err := s.Client.Services.EC2.DeleteSecurityGroup(ctx, &input); err != nil {
		return err
	}

	s.Resource = nil
	return nil
}

func (s *SecurityGroup) generatePermissions(rule universal.FirewallRule) []types.IpPermission {
	var ranges []types.IpRange
	if rule.Nets == nil {
		ranges = append(ranges, types.IpRange{
			CidrIp: aws.String("0.0.0.0/0"),
		})
	} else {
		for _, block := range *rule.Nets {
			ranges = append(ranges, types.IpRange{
				CidrIp: aws.String(block.String()),
			})
		}
	}

	// Allow any traffic between machines in the same security group.
	permissions := []types.IpPermission{
		{
			IpProtocol: aws.String("-1"),
			UserIdGroupPairs: []types.UserIdGroupPair{
				{
					GroupId: s.Resource.GroupId,
				},
			},
		},
	}

	// Allow the specified external traffic.
	if rule.Ports == nil {
		permissions = append(permissions, types.IpPermission{
			IpProtocol: aws.String("-1"),
			IpRanges:   ranges,
		})
	} else {
		for _, port := range *rule.Ports {
			for _, protocol := range []string{"tcp", "udp"} {
				permissions = append(permissions, types.IpPermission{
					IpProtocol: aws.String(protocol),
					FromPort:   aws.Int32(int32(port)),
					ToPort:     aws.Int32(int32(port)),
					IpRanges:   ranges,
				})
			}
		}
	}

	return permissions
}
