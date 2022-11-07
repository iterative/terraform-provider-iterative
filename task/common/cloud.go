package common

import (
	"errors"
	"time"
)

type Cloud struct {
	Timeouts    Timeouts
	Provider    Provider
	Credentials Credentials
	Region      Region
	Tags        map[string]string
}

type Timeouts struct {
	Create time.Duration
	Read   time.Duration
	Update time.Duration
	Delete time.Duration
}

type Region string
type Provider string

const (
	ProviderAWS Provider = "aws"
	ProviderGCP Provider = "gcp"
	ProviderAZ  Provider = "az"
	ProviderK8S Provider = "k8s"
)

type Credentials struct {
	AWSCredentials *AWSCredentials
	GCPCredentials *GCPCredentials
	AZCredentials  *AZCredentials
	K8SCredentials *K8SCredentials
}

type AWSCredentials struct {
	AccessKeyID     string // AWS_ACCESS_KEY_ID
	SecretAccessKey string // AWS_SECRET_ACCESS_KEY
	SessionToken    string // AWS_SESSION_TOKEN
}

type GCPCredentials struct {
	ApplicationCredentials string // GOOGLE_APPLICATION_CREDENTIALS (contents of file)
}

type AZCredentials struct {
	ClientID       string // AZURE_CLIENT_ID
	ClientSecret   string // AZURE_CLIENT_SECRET
	SubscriptionID string // AZURE_SUBSCRIPTION_ID
	TenantID       string // AZURE_TENANT_ID
}

type K8SCredentials struct {
	Config string // KUBECONFIG (contents of file)
}

func (c *Cloud) GetClosestRegion(regions map[string]Region) (string, error) {
	for key, value := range regions {
		if value == c.Region {
			return key, nil
		}
	}

	return "", errors.New("native region not found")
}
