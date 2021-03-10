package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadinessCheck(t *testing.T) {
	t.Run("Runner log containing a readiness message should be considered ready", func(t *testing.T) {
		logString := `
			-- Logs begin at Wed 2021-01-20 00:25:37 UTC, end at Fri 2021-02-26 15:37:56 UTC. --
			Feb 26 15:37:20 ip-172-31-6-188 cml.sh[2203]: {"level":"info","time":"···","repo":"···","status":"ready"}
		`
		assert.True(t, IsReady(logString))
	})

	t.Run("Runner log not containing a readiness message should not be considered ready", func(t *testing.T) {
		logString := `
			-- Logs begin at Wed 2021-01-20 00:25:37 UTC, end at Fri 2021-02-26 15:37:56 UTC. --
			Feb 26 15:37:20 ip-172-31-6-188 cml.sh[2203]: {"level":"info","time":"···","repo":"···"}
		`
		assert.False(t, IsReady(logString))
	})
}
