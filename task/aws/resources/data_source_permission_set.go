package resources

import (
	"context"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/iterative/terraform-provider-iterative/task/aws/client"
)

var validateARN = regexp.MustCompile(`arn:aws:iam::[\d]*:instance-profile/[\S]*`)

func NewPermissionSet(client *client.Client, identifier string) *PermissionSet {
	return &PermissionSet{
		client:     client,
		Identifier: identifier,
	}
}

type PermissionSet struct {
	client     *client.Client
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
	if !validateARN.MatchString(arn) {
		return fmt.Errorf("invalid IAM Instance Profile: %s", arn)
	}
	ps.Resource = &types.LaunchTemplateIamInstanceProfileSpecificationRequest{
		Arn: aws.String(arn),
	}
	return nil
}
