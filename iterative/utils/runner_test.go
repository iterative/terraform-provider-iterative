package utils

import "testing"

func TestPositiveReadinessCheck(t *testing.T) {
	result := IsReady(`
        -- Logs begin at Wed 2021-01-20 00:25:37 UTC, end at Fri 2021-02-26 15:37:56 UTC. --
        Feb 26 15:37:20 ip-172-31-6-188 cml.sh[2203]: {"level":"info","time":"···","repo":"···","status":"ready"}
    `)
	if !result {
		t.Errorf("Positive readiness check")
	}
}

func TestNegativeReadinessCheck(t *testing.T) {
	result := IsReady(`
        -- Logs begin at Wed 2021-01-20 00:25:37 UTC, end at Fri 2021-02-26 15:37:56 UTC. --
        Feb 26 15:37:20 ip-172-31-6-188 cml.sh[2203]: {"level":"info","time":"···","repo":"···"}
    `)
	if result {
		t.Errorf("Negative readiness check")
	}
}
