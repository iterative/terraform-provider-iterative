package common

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
)


func TestIdentifier(t *testing.T) {
	faker := gofakeit.NewCrypto()

	t.Run("idempotency", func(t *testing.T) {
		for count := 0; count < 1000; count++ {
			value := faker.Sentence(512)
			once := Identifier(value).Long()
			twice := Identifier(once).Long()
			require.Equal(t, once, twice)
		}
	})

	t.Run("stability", func(t *testing.T) {
		value := faker.Sentence(512)
		sample := Identifier(value)
		longSample := Identifier(value).Long()
		shortSample := Identifier(value).Short()
		for count := 0; count < 1000; count++ {
			require.Equal(t, longSample, sample.Long())
			require.Equal(t, shortSample, sample.Short())
			require.Equal(t, longSample, Identifier(value).Long())
			require.Equal(t, shortSample, Identifier(value).Short())
		}
	})

	t.Run("homogeneity", func(t *testing.T) {
		for count := 0; count < 1000; count++ {
			value := faker.Sentence(512)

			identifier := Identifier(value)
			long := identifier.Long()
			short := identifier.Short()

			require.Regexp(t, "^tpi-[a-z0-9-]+$", long)
			require.Regexp(t, "^[a-z0-9]+$", short)

			require.LessOrEqual(t, len(long), 50)
			require.LessOrEqual(t, len(short), 24)
		}
	})
}