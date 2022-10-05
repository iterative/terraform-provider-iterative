package resources

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
)

func NewLaunchTemplate(client *client.Client, identifier common.Identifier, securityGroup *SecurityGroup, permissionSet *PermissionSet, image *Image, keyPair *KeyPair, credentials *Credentials, task common.Task) *LaunchTemplate {
	l := &LaunchTemplate{
		client:     client,
		Identifier: identifier.Long(),
		Attributes: task,
	}
	l.Dependencies.SecurityGroup = securityGroup
	l.Dependencies.Image = image
	l.Dependencies.KeyPair = keyPair
	l.Dependencies.Credentials = credentials
	l.Dependencies.PermissionSet = permissionSet
	return l
}

type LaunchTemplate struct {
	client       *client.Client
	Identifier   string
	Attributes   common.Task
	Dependencies struct {
		KeyPair       *KeyPair
		SecurityGroup *SecurityGroup
		Image         *Image
		Credentials   *Credentials
		PermissionSet *PermissionSet
	}
	Resource *types.LaunchTemplate
}

func (l *LaunchTemplate) Create(ctx context.Context) error {
	if l.Attributes.Environment.Variables == nil {
		l.Attributes.Environment.Variables = make(map[string]*string)
	}

	timeout := time.Now().Add(l.Attributes.Environment.Timeout)
	script, err := machine.Script(l.Attributes.Environment.Script, l.Dependencies.Credentials.Resource, l.Attributes.Environment.Variables, &timeout)
	if err != nil {
		return fmt.Errorf("failed to render machine script: %w", err)
	}
	userData := base64.StdEncoding.EncodeToString([]byte(script))

	size := l.Attributes.Size.Machine
	sizes := map[string]string{
		"s":       "t2.micro",
		"m":       "m5.2xlarge",
		"l":       "m5.8xlarge",
		"xl":      "m5.16xlarge",
		"m+t4":    "g4dn.xlarge",
		"m+k80":   "p2.xlarge",
		"l+k80":   "p2.8xlarge",
		"xl+k80":  "p2.16xlarge",
		"m+v100":  "p3.xlarge",
		"l+v100":  "p3.8xlarge",
		"xl+v100": "p3.16xlarge",
	}

	if val, ok := sizes[size]; ok {
		size = val
	}

	input := ec2.CreateLaunchTemplateInput{
		LaunchTemplateName: aws.String(l.Identifier),
		LaunchTemplateData: &types.RequestLaunchTemplateData{
			UserData:           aws.String(userData),
			ImageId:            l.Dependencies.Image.Resource.ImageId,
			KeyName:            l.Dependencies.KeyPair.Resource.KeyName,
			InstanceType:       types.InstanceType(size),
			SecurityGroupIds:   []string{aws.ToString(l.Dependencies.SecurityGroup.Resource.GroupId)},
			IamInstanceProfile: l.Dependencies.PermissionSet.Resource,
			BlockDeviceMappings: []types.LaunchTemplateBlockDeviceMappingRequest{
				{
					DeviceName: aws.String("/dev/sda1"),
					Ebs: &types.LaunchTemplateEbsBlockDeviceRequest{
						DeleteOnTermination: aws.Bool(true),
						Encrypted:           aws.Bool(false),
						VolumeType:          types.VolumeType("gp2"),
					},
				},
			},
			TagSpecifications: []types.LaunchTemplateTagSpecificationRequest{
				{
					ResourceType: types.ResourceTypeInstance,
					Tags:         makeTagSlice(l.Identifier, l.client.Tags),
				},
				{
					ResourceType: types.ResourceTypeVolume,
					Tags:         makeTagSlice(l.Identifier, l.client.Tags),
				},
			},
		},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeLaunchTemplate,
				Tags:         makeTagSlice(l.Identifier, l.client.Tags),
			},
		},
	}

	if size := l.Attributes.Size.Storage; size > 0 {
		input.LaunchTemplateData.BlockDeviceMappings[0].Ebs.VolumeSize = aws.Int32(int32(size))
	}

	if _, err = l.client.Services.EC2.CreateLaunchTemplate(ctx, &input); err != nil {
		var e smithy.APIError
		if errors.As(err, &e) && e.ErrorCode() == "InvalidLaunchTemplateName.AlreadyExistsException" {
			return l.Read(ctx)
		}
		return err
	}

	return l.Read(ctx)
}

func (l *LaunchTemplate) Read(ctx context.Context) error {
	input := ec2.DescribeLaunchTemplatesInput{
		LaunchTemplateNames: []string{l.Identifier},
	}

	templates, err := l.client.Services.EC2.DescribeLaunchTemplates(ctx, &input)
	if err != nil {
		return err
	}

	if len(templates.LaunchTemplates) == 0 {
		return common.NotFoundError
	}

	l.Resource = &templates.LaunchTemplates[0]
	return nil
}

func (l *LaunchTemplate) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (l *LaunchTemplate) Delete(ctx context.Context) error {
	input := ec2.DeleteLaunchTemplateInput{
		LaunchTemplateName: aws.String(l.Identifier),
	}

	if _, err := l.client.Services.EC2.DeleteLaunchTemplate(ctx, &input); err != nil {
		var e smithy.APIError
		if errors.As(err, &e) && e.ErrorCode() == "InvalidLaunchTemplateName.NotFoundException" {
			l.Resource = nil
			return nil
		}
		return err
	}

	l.Resource = nil
	return nil
}
