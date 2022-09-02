package common

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"crypto/sha256"

	"github.com/aohorodnyk/uid"
)

type Identifier struct {
	name string
	salt string
}

var ErrWrongIdentifier = errors.New("wrong identifier")

const (
	maximumLongLength = 50
	shortLength       = 16
	nameLength        = maximumLongLength-shortLength-uint32(len("tpi---"))
)

func ParseIdentifier(identifier string) (Identifier, error) {
	re := regexp.MustCompile(`(?s)^tpi-([a-z0-9]+(?:[a-z0-9-]*[a-z0-9])?)-([a-z0-9]+)-([a-z0-9]+)$`)

	if match := re.FindStringSubmatch(string(identifier)); len(match) > 0 && hash(match[1]+match[2], shortLength/2) == match[3] {
		return Identifier{name: match[1], salt: match[2]}, nil
	}

	return Identifier{}, ErrWrongIdentifier
}

func NewDeterministicIdentifier(name string) Identifier {
	seed := normalize(name, nameLength)
	return Identifier{name: name, salt: hash(seed, shortLength/2)}
}

func NewRandomIdentifier(name string) Identifier {
	seed := uid.NewProvider36Size(8).MustGenerate().String()
	if name == "" {
		name = seed
	}

	return Identifier{name: name, salt: hash(seed, shortLength/2)}
}

func (i Identifier) Long() string {
	name := normalize(i.name, nameLength)
	return fmt.Sprintf("tpi-%s-%s-%s", name, i.salt, hash(name+i.salt, shortLength/2))
}

func (i Identifier) Short() string {
	p := strings.Split(i.Long(), "-")
	return p[len(p)-2] + p[len(p)-1]
}

// hash deterministically generates a Base36 digest of `size`
// characters using the provided salt.
func hash(salt string, size uint8) string {
	digest := sha256.Sum256([]byte(salt))
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
