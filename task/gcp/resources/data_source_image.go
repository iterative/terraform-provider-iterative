package resources

import (
	"context"
	"errors"
	"regexp"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/gcp/client"
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
	Resource *compute.Image
}

func (i *Image) Read(ctx context.Context) error {
	image := i.Identifier
	images := map[string]string{
		"ubuntu": "ubuntu@ubuntu-os-cloud/ubuntu-2004-lts",
	}
	if val, ok := images[image]; ok {
		image = val
	}

	match := regexp.MustCompile(`^([^@]+)@([^/]+)/([^/]+)$`).FindStringSubmatch(image)
	if match == nil {
		return common.NotFoundError
	}

	i.Attributes.SSHUser = match[1]
	project := match[2]
	family := match[3]

	resource, err := i.Client.Services.Compute.Images.GetFromFamily(project, family).Do()
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
