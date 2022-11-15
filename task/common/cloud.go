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

// Credentials define the cloud provider credentials.
type Credentials struct {
	AWSCredentials *AWSCredentials `json:"aws,omitempty"`
	GCPCredentials *GCPCredentials `json:"gcp,omitempty"`
	AZCredentials  *AZCredentials  `json:"az,omitempty"`
	K8SCredentials *K8SCredentials `json:"k8s,omitempty"`
}

type AWSCredentials struct {
	AccessKeyID     string `json:"access-key-id,omitempty"`     // AWS_ACCESS_KEY_ID
	SecretAccessKey string `json:"secret-access-key,omitempty"` // AWS_SECRET_ACCESS_KEY
	SessionToken    string `json:"session-token,omitempty"`     // AWS_SESSION_TOKEN
}

type GCPCredentials struct {
	ApplicationCredentials string `json:"credentials,omitempty"` // GOOGLE_APPLICATION_CREDENTIALS (contents of file)
}

type AZCredentials struct {
	ClientID       string `json:"client-id,omitempty"`       // AZURE_CLIENT_ID
	ClientSecret   string `json:"client-secret,omitempty"`   // AZURE_CLIENT_SECRET
	SubscriptionID string `json:"subscription-id,omitempty"` // AZURE_SUBSCRIPTION_ID
	TenantID       string `json:"tenant-id,omitempty"`       // AZURE_TENANT_ID
}

type K8SCredentials struct {
	Config string `json:"config,omitempty"` // KUBECONFIG (contents of file)
}

func (c *Cloud) GetClosestRegion(regions map[string]Region) (string, error) {
	for key, value := range regions {
		if value == c.Region {
			return key, nil
		}
	}

	return "", errors.New("native region not found")
}
