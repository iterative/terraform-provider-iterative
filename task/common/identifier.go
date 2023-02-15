package common

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"crypto/sha256"

	"github.com/aohorodnyk/uid"
	petname "github.com/dustinkirkland/golang-petname"
)

type Identifier struct {
	prefix string
	name   string
	salt   string
}

var ErrWrongIdentifier = errors.New("wrong identifier")

const (
	defaultIdentifierPrefix = "tpi"
	maximumLongLength       = 50
	shortLength             = 16
	nameLength              = maximumLongLength - shortLength - uint32(len("tpi---"))
)

// ParseIdentifier parses the string representation of the identifier.
func ParseIdentifier(identifier string) (Identifier, error) {
	re := regexp.MustCompile(`(?s)^([a-z0-9]{3})-([a-z0-9]+(?:[a-z0-9-]*[a-z0-9])?)-([a-z0-9]+)-([a-z0-9]+)$`)

	if match := re.FindStringSubmatch(string(identifier)); len(match) > 0 && hash(match[2]+match[3], shortLength/2) == match[4] {
		return Identifier{prefix: match[1], name: match[2], salt: match[3]}, nil
	}

	return Identifier{}, ErrWrongIdentifier
}

// NewDeterministicIdentifierWithPrefix returns a new deterministic Identifier, with
// the specified prefix, using the provided name as a seed. Repeated calls to this
// function are guaranteed to generate the same Identifier.
func NewDeterministicIdentifierWithPrefix(prefix, name string) Identifier {
	seed := normalize(name, nameLength)
	return Identifier{prefix: prefix[0:3], name: name, salt: hash(seed, shortLength/2)}
}

// NewDeterministicIdentifier returns a new deterministic Identifier, using the
// provided name as a seed. Repeated calls to this function are guaranteed to
// generate the same Identifier.
func NewDeterministicIdentifier(name string) Identifier {
	return NewDeterministicIdentifierWithPrefix(defaultIdentifierPrefix, name)
}

// NewRandomIdentifierWithPrefix returns a new random Identifier with the
// specified prefix. Only the first 3 symbols of the prefix are used.
// Repeated calls to this function are guaranteed to generate different
// Identifiers, as long as there are no collisions.
func NewRandomIdentifierWithPrefix(prefix, name string) Identifier {
	seed := uid.NewProvider36Size(8).MustGenerate().String()
	if name == "" {
		petname.NonDeterministicMode()
		name = petname.Generate(3, "-")
	}
	return Identifier{prefix: prefix[0:3], name: name, salt: hash(seed, shortLength/2)}
}

// NewRandomIdentifier returns a new random Identifier.
// Repeated calls to this function are guaranteed to generate different
// Identifiers, as long as there are no collisions.
func NewRandomIdentifier(name string) Identifier {
	return NewRandomIdentifierWithPrefix(defaultIdentifierPrefix, name)
}

func (i Identifier) Long() string {
	name := normalize(i.name, nameLength)
	return fmt.Sprintf("%s-%s-%s-%s", i.prefix, name, i.salt, hash(name+i.salt, shortLength/2))
}

func (i Identifier) Short() string {
	p := strings.Split(i.Long(), "-")
	return p[len(p)-2] + p[len(p)-1]
}

// hash deterministically generates a Base36 digest of `size`
// characters using the provided seed.
func hash(seed string, size uint8) string {
	digest := sha256.Sum256([]byte(seed))
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
