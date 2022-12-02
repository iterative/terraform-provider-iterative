package resources

import (
	"context"
	"errors"
	"regexp"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"github.com/iterative/terraform-provider-iterative/task/common"
	"github.com/iterative/terraform-provider-iterative/task/gcp/client"
)

func NewImage(client *client.Client, identifier string) *Image {
	return &Image{
		client:     client,
		Identifier: identifier,
	}
}

type Image struct {
	client     *client.Client
	Identifier string
	Attributes struct {
		SSHUser string
	}
	Resource *compute.Image
}

func (i *Image) Read(ctx context.Context) error {
	// default image to ubuntu in not present
	if i.Identifier == "" {
		i.Identifier = "ubuntu"
	}
	image := i.Identifier
	images := map[string]string{
		"ubuntu": "ubuntu@ubuntu-os-cloud/ubuntu-2004-lts",
		"nvidia": "ubuntu@deeplearning-platform-release/common-cu113-ubuntu-2004",
	}
	if val, ok := images[image]; ok {
		image = val
	}

	match := regexp.MustCompile(`^([^@]+)@([^/]+)/([^/]+)$`).FindStringSubmatch(image)
	if match == nil {
		return errors.New("wrong image name")
	}

	i.Attributes.SSHUser = match[1]
	project := match[2]
	imageOrFamily := match[3]

	resource, err := i.client.Services.Compute.Images.Get(project, imageOrFamily).Do()
	if err != nil {
		var e *googleapi.Error
		if errors.As(err, &e) && e.Code == 404 {
			resource, err := i.client.Services.Compute.Images.GetFromFamily(project, imageOrFamily).Do()
			if err != nil {
				var e *googleapi.Error
				if errors.As(err, &e) && e.Code == 404 {
					return common.NotFoundError
				}
				return err
			}
			i.Resource = resource
			return nil
		}
		return err
	}

	i.Resource = resource
	return nil
}
