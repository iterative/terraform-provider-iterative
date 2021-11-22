package task

import (
	"context"
	"os"
	"testing"
	"time"

	"io/ioutil"
	"path/filepath"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"terraform-provider-iterative/task/common"
)

func TestTask(t *testing.T) {
	if testing.Short() {
		t.Skip("go test -short detected, skipping smoke tests")
	}

	testIdentifier := "smoke test"
	if custom := os.Getenv("SMOKE_TEST_IDENTIFIER"); custom != "" {
		testIdentifier = custom
	}
	sweepOnly := os.Getenv("SMOKE_TEST_SWEEP") != ""

	providers := []common.Provider{
		common.ProviderAWS,
		common.ProviderAZ,
		common.ProviderGCP,
		common.ProviderK8S,
	}

	for _, provider := range providers {
		t.Run(string(provider), func(t *testing.T) {
			oldData := gofakeit.Phrase()
			newData := gofakeit.Phrase()

			dataDirectory := t.TempDir()
			dataFile := filepath.Join(dataDirectory, "data")

			cloud := common.Cloud{
				Provider: provider,
				Region:   common.Region("us-west"),
				Timeouts: common.Timeouts{
					Create: 10 * time.Minute,
					Read:   10 * time.Minute,
					Update: 10 * time.Minute,
					Delete: 10 * time.Minute,
				},
			}

			identifier := common.Identifier(testIdentifier)

			task := common.Task{
				Size: common.Size{
					Machine: "m",
					Storage: 30,
				},
				Environment: common.Environment{
					Image: "ubuntu",
					Script: `#!/bin/bash
						cat data
						echo "$ENVIRONMENT_VARIABLE_DATA" | tee data
					`,
					Variables: map[string]*string{
						"ENVIRONMENT_VARIABLE_DATA": &newData,
					},
					Directory: dataDirectory,
					Timeout:   10 * time.Minute,
				},
				Firewall: common.Firewall{
					Ingress: common.FirewallRule{
						Ports: &[]uint16{22},
					},
					// Egress: everything open.
				},
				Spot:        common.SpotEnabled,
				Parallelism: 1,
			}

			ctx := context.TODO()

			newTask, err := New(ctx, cloud, identifier, task)
			require.Nil(t, err)

			require.Nil(t, newTask.Delete(ctx))
			if sweepOnly {
				return
			}

			file, err := os.Create(dataFile)
			require.Nil(t, err)

			_, err = file.WriteString(oldData)
			require.Nil(t, err)
			require.Nil(t, file.Close())

			require.Nil(t, newTask.Create(ctx))
			require.Nil(t, newTask.Create(ctx))

		loop:
			for assert.Nil(t, newTask.Read(ctx)) {
				logs, err := newTask.Logs(ctx)
				require.Nil(t, err)

				for _, log := range logs {
					if assert.Contains(t, log, oldData) &&
						assert.Contains(t, log, newData) {
						break loop
					}
				}
			}

			if provider == common.ProviderK8S {
				require.Equal(t, newTask.Start(ctx), common.NotImplementedError)
				require.Equal(t, newTask.Stop(ctx), common.NotImplementedError)
			} else {
				require.Nil(t, newTask.Stop(ctx))
				require.Nil(t, newTask.Stop(ctx))

				for assert.Nil(t, newTask.Read(ctx)) &&
					newTask.Status(ctx)[common.StatusCodeRunning] > 0 {
					continue
				}

				require.Nil(t, newTask.Start(ctx))
				require.Nil(t, newTask.Start(ctx))

				for assert.Nil(t, newTask.Read(ctx)) &&
					newTask.Status(ctx)[common.StatusCodeRunning] == 0 {
					continue
				}
			}

			require.Nil(t, newTask.Delete(ctx))
			require.Nil(t, newTask.Delete(ctx))

			require.FileExists(t, dataFile)

			contents, err := ioutil.ReadFile(dataFile)
			require.Nil(t, err)

			require.Contains(t, string(contents), newData)
		})
	}
}
