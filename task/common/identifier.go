package common

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"crypto/sha256"

	"github.com/aohorodnyk/uid"
)

type Identifier string

const (
	maximumLongLength = 50
	shortLength       = 16
)

func (i Identifier) Long() string {
	re := regexp.MustCompile(`(?s)^tpi-([a-z0-9]+(?:[a-z0-9-]*[a-z0-9])?)-([a-z0-9]+)-([a-z0-9]+)$`)

	if match := re.FindStringSubmatch(string(i)); len(match) > 0 && hash(match[1]+match[2], shortLength/2) == match[3] {
		return match[0]
	}

	name := normalize(string(i), maximumLongLength-shortLength-uint32(len("tpi---")))
	digest := hash(string(i), shortLength/2)

	return fmt.Sprintf("tpi-%s-%s-%s", name, digest, hash(name+digest, shortLength/2))
}

func (i Identifier) Short() string {
	p := strings.Split(i.Long(), "-")
	return p[len(p)-2] + p[len(p)-1]
}

// hash deterministically generates a Base36 digest of `size`
// characters using `identifier` as the seed.
func hash(identifier string, size uint8) string {
	digest := sha256.Sum256([]byte(identifier))
	random := uid.NewRandCustom(bytes.NewReader(digest[:]))
	encoder := uid.NewEncoderBase36()
	provider := uid.NewProviderCustom(sha256.Size, random, encoder)
	result := provider.MustGenerate().String()

	if len(result) < int(size) {
		panic("not enough bytes to satisfy requested size")
	}

	return result[:size]
}

// normalize normalizes user-provided identifiers by adapting them to
// RFC1123-like names truncated to the specified length.
func normalize(identifier string, truncate uint32) string {
	lowercase := strings.ToLower(identifier)

	normalized := regexp.MustCompile("[^a-z0-9]+").ReplaceAllString(lowercase, "-")

	if len(normalized) > int(truncate) {
		normalized = normalized[:truncate]
	}

	return regexp.MustCompile("(^-)|(-$)").ReplaceAllString(normalized, "")
}
