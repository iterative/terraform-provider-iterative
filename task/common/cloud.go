package common

import (
	"errors"
	"time"
)

type Cloud struct {
	Timeouts
	Provider
	Region
	Tags map[string]string
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

func (c *Cloud) GetClosestRegion(regions map[string]Region) (string, error) {
	for key, value := range regions {
		if value == c.Region {
			return key, nil
		}
	}

	return "", errors.New("native region not found")
}
