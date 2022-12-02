package client

import (
	"context"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/iterative/terraform-provider-iterative/task/common"
	"github.com/iterative/terraform-provider-iterative/task/common/ssh"
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

	options := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}

	if awsCredentials := cloud.Credentials.AWSCredentials; awsCredentials != nil {
		options = append(options, config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     awsCredentials.AccessKeyID,
				SecretAccessKey: awsCredentials.SecretAccessKey,
				SessionToken:    awsCredentials.SessionToken,
				Source:          "user-specified credentials",
			},
		}))
	}

	config, err := config.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return nil, err
	}
	credentials, err := config.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, err
	}
	c := &Client{
		Cloud:       cloud,
		Region:      region,
		Tags:        cloud.Tags,
		Config:      config,
		credentials: credentials,
	}

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

	Config      aws.Config
	credentials aws.Credentials
	Services    struct {
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

// Credentials returns the AWS credentials the client is currently using.
func (c *Client) Credentials() aws.Credentials {
	return c.credentials
}
