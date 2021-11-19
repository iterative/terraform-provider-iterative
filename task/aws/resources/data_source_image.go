package resources

import (
	"context"
	"regexp"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/common"
)

func NewImage(client *client.Client, identifier string) *Image {
	i := new(Image)
	i.Client = client
	i.Identifier = identifier
	return i
}

type Image struct {
	Client     *client.Client
	Identifier string
	Attributes struct {
		SSHUser string
	}
	Resource *types.Image
}

func (i *Image) Read(ctx context.Context) error {
	image := i.Identifier
	images := map[string]string{
		"ubuntu": "ubuntu@099720109477:x86_64:*ubuntu/images/hvm-ssd/ubuntu-focal-20.04*",
	}
	if val, ok := images[image]; ok {
		image = val
	}

	match := regexp.MustCompile(`^([^@]+)@([^:]+):([^:]+):([^:]+)$`).FindStringSubmatch(image)
	if match == nil {
		return common.NotFoundError
	}

	i.Attributes.SSHUser = match[1]
	owner := match[2]
	architecture := match[3]
	name := match[4]

	filters := []types.Filter{
		{
			Name:   aws.String("name"),
			Values: []string{name},
		},
	}
	if architecture != "*" {
		filters = append(filters, types.Filter{
			Name:   aws.String("architecture"),
			Values: []string{architecture},
		})
	}
	if owner != "*" {
		filters = append(filters, types.Filter{
			Name:   aws.String("owner-id"),
			Values: []string{owner},
		})
	}

	input := ec2.DescribeImagesInput{
		Filters: filters,
	}

	result, err := i.Client.Services.EC2.DescribeImages(ctx, &input)
	if err != nil {
		return err
	}

	sort.Slice(result.Images, func(a, b int) bool {
		timeA, err := time.Parse(time.RFC3339, aws.ToString(result.Images[a].CreationDate))
		if err != nil {
			panic(err)
		}
		timeB, err := time.Parse(time.RFC3339, aws.ToString(result.Images[b].CreationDate))
		if err != nil {
			panic(err)
		}
		return timeA.Unix() > timeB.Unix()
	})

	if len(result.Images) == 0 {
		return common.NotFoundError
	}

	i.Resource = &result.Images[0]
	return nil
}
