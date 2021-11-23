package common

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"crypto/sha256"

	"github.com/aohorodnyk/uid"
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

// NormalizeIdentifier normalizes user-provided identifiers by adapting them to
// the most strict cloud provider naming requisites.
func NormalizeIdentifier(identifier string, long bool) string {
	lowercase := strings.ToLower(identifier)
	normalized := regexp.MustCompile("[^a-z0-9]+").ReplaceAllString(lowercase, "-")
	normalized = regexp.MustCompile("(^-)|(-$)").ReplaceAllString(normalized, "")
	if len(normalized) > 50 {
		normalized = normalized[:50]
	}
	if long {
		return fmt.Sprintf("tpi-%s-%s", normalized, generateSuffix(identifier, 8))
	}
	return generateSuffix(identifier, 8)
}

// generateSuffix deterministically generates a Base36 suffix of `size`
// characters using `identifier` as the seed. This is useful to enhance
// user-provided identifiers so they become valid "globally unique names" for
// AWS S3, Google Storage and other services that require them.
func generateSuffix(identifier string, size uint32) string {
	digest := sha256.Sum256([]byte(identifier))
	random := uid.NewRandCustom(bytes.NewReader(digest[:]))
	encoder := uid.NewEncoderBase36()
	provider := uid.NewProviderCustom(size, random, encoder)
	return provider.MustGenerate().String()
}
