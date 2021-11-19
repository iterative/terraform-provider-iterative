package resources

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/aws/smithy-go"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/machine"
)

func NewLaunchTemplate(client *client.Client, identifier common.Identifier, securityGroup *SecurityGroup, image *Image, keyPair *KeyPair, credentials *Credentials, task common.Task) *LaunchTemplate {
	l := new(LaunchTemplate)
	l.Client = client
	l.Identifier = identifier.Long()
	l.Attributes = task
	l.Dependencies.SecurityGroup = securityGroup
	l.Dependencies.Image = image
	l.Dependencies.KeyPair = keyPair
	l.Dependencies.Credentials = credentials
	return l
}

type LaunchTemplate struct {
	Client       *client.Client
	Identifier   string
	Attributes   common.Task
	Dependencies struct {
		*KeyPair
		*SecurityGroup
		*Image
		*Credentials
	}
	Resource *types.LaunchTemplate
}

func (l *LaunchTemplate) Create(ctx context.Context) error {
	if l.Attributes.Environment.Variables == nil {
		l.Attributes.Environment.Variables = make(map[string]*string)
	}
	for name, value := range *l.Dependencies.Credentials.Resource {
		valueCopy := value
		l.Attributes.Environment.Variables[name] = &valueCopy
	}

	script := machine.Script(l.Attributes.Environment.Script, l.Attributes.Environment.Variables, l.Attributes.Environment.Timeout)
	userData := base64.StdEncoding.EncodeToString([]byte(script))

	size := l.Attributes.Size.Machine
	sizes := map[string]string{
		"m":       "m5.2xlarge",
		"l":       "m5.8xlarge",
		"xl":      "m5.16xlarge",
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
			UserData:         aws.String(userData),
			ImageId:          l.Dependencies.Image.Resource.ImageId,
			KeyName:          l.Dependencies.KeyPair.Resource.KeyName,
			InstanceType:     types.InstanceType(size),
			SecurityGroupIds: []string{aws.ToString(l.Dependencies.SecurityGroup.Resource.GroupId)},
			BlockDeviceMappings: []types.LaunchTemplateBlockDeviceMappingRequest{
				{
					DeviceName: aws.String("/dev/sda1"),
					Ebs: &types.LaunchTemplateEbsBlockDeviceRequest{
						DeleteOnTermination: aws.Bool(true),
						Encrypted:           aws.Bool(false),
						VolumeSize:          aws.Int32(int32(l.Attributes.Size.Storage)),
						VolumeType:          types.VolumeType("gp2"),
					},
				},
			},
			TagSpecifications: []types.LaunchTemplateTagSpecificationRequest{
				{
					ResourceType: types.ResourceTypeInstance,
					Tags:         makeTagSlice(l.Identifier, l.Client.Tags),
				},
				{
					ResourceType: types.ResourceTypeVolume,
					Tags:         makeTagSlice(l.Identifier, l.Client.Tags),
				},
			},
		},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeLaunchTemplate,
				Tags:         makeTagSlice(l.Identifier, l.Client.Tags),
			},
		},
	}

	if _, err := l.Client.Services.EC2.CreateLaunchTemplate(ctx, &input); err != nil {
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

	templates, err := l.Client.Services.EC2.DescribeLaunchTemplates(ctx, &input)
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

	if _, err := l.Client.Services.EC2.DeleteLaunchTemplate(ctx, &input); err != nil {
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
