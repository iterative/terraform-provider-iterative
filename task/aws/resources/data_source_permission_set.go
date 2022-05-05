package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

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
	arn := ps.Identifier
	// "", "arn:*"
	if arn == "" {
		ps.Resource = nil
		return nil
	}
	if strings.HasPrefix(arn, "arn:aws:iam:") {
		ps.Resource = &types.LaunchTemplateIamInstanceProfileSpecificationRequest{
			Arn: aws.String(arn),
		}
		return nil
	}
	return fmt.Errorf("invlaid IAM Instance Profile: %s", arn)
}
