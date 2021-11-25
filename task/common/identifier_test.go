package common

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
)


func TestIdentifier(t *testing.T) {
	name := gofakeit.NewCrypto().Sentence(512)

	t.Run("idempotence", func(t *testing.T) {
		identifier := Identifier(name)

		once := identifier.Long()
		twice := Identifier(once).Long()
		require.Equal(t, once, twice)
	})

	t.Run("stability", func(t *testing.T) {
		identifier := Identifier(name)

		require.Equal(t, identifier.Long(), identifier.Long())
		require.Equal(t, identifier.Short(), identifier.Short())
	})

	t.Run("homogeneity", func(t *testing.T) {
		identifier := Identifier(name)

		long := identifier.Long()
		short := identifier.Short()

		require.Regexp(t, "^tpi-[a-z0-9-]+$", long)
		require.Regexp(t, "^[a-z0-9]+$", short)

		require.LessOrEqual(t, len(long), 50)
		require.LessOrEqual(t, len(short), 24)
	})

	t.Run("compatibility", func(t *testing.T) {
		identifier := Identifier("test")

		require.Equal(t, identifier.Long(), "tpi-test-189gt4x-1q5wad0")
		require.Equal(t, identifier.Short(), "189gt4x1q5wad0")
	})
}