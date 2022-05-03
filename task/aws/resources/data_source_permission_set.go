package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"

	"terraform-provider-iterative/task/aws/client"
)

func NewPermissionSet(client *client.Client, identifier string) *PermissionSet {
	ps := new(PermissionSet)
	ps.Client = client
	ps.Identifier = identifier
	return ps
}

type PermissionSet struct {
	Client     *client.Client
	Identifier string
	Resource   *types.LaunchTemplateIamInstanceProfileSpecificationRequest
}

func (ps *PermissionSet) Read(ctx context.Context) error {
	nameOrArn := ps.Identifier
	// "", "arn:*", "name"
	if nameOrArn == "" {
		ps.Resource = nil
		return nil
	}
	if strings.HasPrefix(nameOrArn, "arn:") {
		ps.Resource = &types.LaunchTemplateIamInstanceProfileSpecificationRequest{
			Arn: aws.String(nameOrArn),
		}
		return nil
	}
	input := &iam.GetInstanceProfileInput{
		InstanceProfileName: &nameOrArn,
	}
	resp, err := ps.Client.Services.IAM.GetInstanceProfile(ctx, input)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("no IAM Instance Profile with name %s", nameOrArn)
	}
	ps.Resource = &types.LaunchTemplateIamInstanceProfileSpecificationRequest{
		Arn: resp.InstanceProfile.Arn,
	}
	return nil
}
