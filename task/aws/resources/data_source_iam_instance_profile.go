package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"

	"terraform-provider-iterative/task/aws/client"
)

func NewIamInstanceProfile(client *client.Client, identifier string) *IamInstanceProfile {
	iip := new(IamInstanceProfile)
	iip.Client = client
	iip.Identifier = identifier
	return iip
}

type IamInstanceProfile struct {
	Client     *client.Client
	Identifier string
	Resource   *types.LaunchTemplateIamInstanceProfileSpecificationRequest
}

func (iip *IamInstanceProfile) Read(ctx context.Context) error {
	nameOrArn := iip.Identifier
	// "", "arn:*", "name"
	if nameOrArn == "" {
		iip.Resource = nil
		return nil
	}
	if strings.HasPrefix(nameOrArn, "arn:") {
		iip.Resource = &types.LaunchTemplateIamInstanceProfileSpecificationRequest{
			Arn: &nameOrArn,
		}
		return nil
	}
	input := &iam.GetInstanceProfileInput{
		InstanceProfileName: &nameOrArn,
	}
	resp, err := iip.Client.Services.IAM.GetInstanceProfile(ctx, input)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("no IAM Instance Profile with name %s", nameOrArn)
	}
	iip.Resource = &types.LaunchTemplateIamInstanceProfileSpecificationRequest{
		Arn: resp.InstanceProfile.Arn,
	}
	return nil
}
