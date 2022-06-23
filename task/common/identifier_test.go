package common

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
)

func TestIdentifier(t *testing.T) {
	name := gofakeit.NewCrypto().Sentence(512)

	t.Run("stability", func(t *testing.T) {
		identifier := NewIdentifier(name)

		require.Equal(t, identifier.Long(), identifier.Long())
		require.Equal(t, identifier.Short(), identifier.Short())
	})

	t.Run("homogeneity", func(t *testing.T) {
		identifier := NewIdentifier(name)

		long := identifier.Long()
		short := identifier.Short()

		require.Regexp(t, "^tpi-[a-z0-9-]+$", long)
		require.Regexp(t, "^[a-z0-9]+$", short)

		require.LessOrEqual(t, len(long), maximumLongLength)
		require.Equal(t, len(short), shortLength)
	})

	t.Run("compatibility", func(t *testing.T) {
		identifier := NewIdentifier("test")

		require.Equal(t, "tpi-test-3z4xlzwq-3u0vweb4", identifier.Long())
		require.Equal(t, "3z4xlzwq3u0vweb4", identifier.Short())

		parsed, err := ParseIdentifier(identifier.Long())
		require.Equal(t, parsed, identifier)
		require.NoError(t, err)
	})
}
