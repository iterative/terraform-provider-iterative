package client

import (
	"context"
	"errors"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/storage/v1"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/ssh"
)

func New(ctx context.Context, cloud common.Cloud, tags map[string]string) (*Client, error) {
	scopes := []string{
		compute.ComputeScope,
		storage.DevstorageReadWriteScope,
	}

	credentialsData := []byte(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_DATA"))

	var err error
	var credentials *google.Credentials
	if len(credentialsData) > 0 {
		credentials, err = google.CredentialsFromJSON(ctx, credentialsData, scopes...)
	} else {
		credentials, err = google.FindDefaultCredentials(ctx, scopes...)
	}

	if err != nil {
		return nil, err
	}

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS_DATA", string(credentials.JSON))

	client := oauth2.NewClient(ctx, credentials.TokenSource)

	region := string(cloud.Region)
	regions := map[string]string{
		"us-east":  "us-east1-c",
		"us-west":  "us-west1-b",
		"eu-north": "europe-north1-a",
		"eu-west":  "europe-west1-d",
	}

	if val, ok := regions[region]; ok {
		region = val
	}

	c := new(Client)
	c.Cloud = cloud
	c.Region = region

	c.Credentials = credentials

	computeService, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	c.Services.Compute = computeService

	storageService, err := storage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	c.Services.Storage = storageService

	c.Identifier = "tpi"
	return c, nil
}

type Client struct {
	Cloud       common.Cloud
	Region      string
	Tags        map[string]string
	Identifier  string
	Credentials *google.Credentials
	Services    struct {
		Compute *compute.Service
		Storage *storage.Service
	}
}

func (c *Client) GetKeyPair(ctx context.Context) (*ssh.DeterministicSSHKeyPair, error) {
	if len(c.Credentials.JSON) == 0 {
		return nil, errors.New("unable to find credentials JSON string")
	}

	return ssh.NewDeterministicSSHKeyPair(string(c.Credentials.JSON), c.Credentials.ProjectID)
}
