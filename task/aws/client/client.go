package client

import (
	"context"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/ssh"
)

func New(ctx context.Context, cloud common.Cloud, tags map[string]string) (*Client, error) {
	region := string(cloud.Region)
	regions := map[string]string{
		"us-east":  "us-east-1",
		"us-west":  "us-west-1",
		"eu-north": "eu-north-1",
		"eu-west":  "eu-west-1",
	}

	if val, ok := regions[region]; ok {
		region = val
	}

	config, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	c := new(Client)
	c.Cloud = cloud
	c.Region = region
	c.Tags = cloud.Tags

	c.Config = config

	c.Services.EC2 = ec2.NewFromConfig(config)
	c.Services.S3 = s3.NewFromConfig(config)
	c.Services.STS = sts.NewFromConfig(config)
	c.Services.AutoScaling = autoscaling.NewFromConfig(config)
	return c, nil
}

type Client struct {
	Cloud  common.Cloud
	Region string
	Tags   map[string]string

	Config   aws.Config
	Services struct {
		EC2         *ec2.Client
		S3          *s3.Client
		STS         *sts.Client
		AutoScaling *autoscaling.Client
	}
}

func (c *Client) GetKeyPair(ctx context.Context) (*ssh.DeterministicSSHKeyPair, error) {
	credentials, err := c.Config.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, err
	}

	return ssh.NewDeterministicSSHKeyPair(credentials.SecretAccessKey, credentials.AccessKeyID)
}

func (c *Client) DecodeError(ctx context.Context, encoded error) error {
	pattern := `(?i)(.*) Encoded authorization failure message: ([\w-]+) ?( .*)?`
	groups := regexp.MustCompile(pattern).FindStringSubmatch(encoded.Error())
	if len(groups) > 2 {
		input := sts.DecodeAuthorizationMessageInput{
			EncodedMessage: aws.String(groups[2]),
		}

		decoded, err := c.Services.STS.DecodeAuthorizationMessage(ctx, &input)
		if err != nil {
			return err
		}

		return fmt.Errorf(
			"%s Authorization failure message: '%s'%s",
			groups[1],
			aws.ToString(decoded.DecodedMessage),
			groups[3],
		)
	}

	return fmt.Errorf("unable to decode: %s", encoded.Error())
}
