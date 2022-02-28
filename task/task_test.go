package task

import (
	"context"
	"os"
	"strings"
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

	testIdentifier := os.Getenv("SMOKE_TEST_IDENTIFIER")
	sweepOnly := os.Getenv("SMOKE_TEST_SWEEP") != ""

	enableAWS := os.Getenv("SMOKE_TEST_ENABLE_AWS") != ""
	enableAZ := os.Getenv("SMOKE_TEST_ENABLE_AZ") != ""
	enableGCP := os.Getenv("SMOKE_TEST_ENABLE_GCP") != ""
	enableK8S := os.Getenv("SMOKE_TEST_ENABLE_K8S") != ""

	enableALL := !enableAWS && !enableAZ && !enableGCP && !enableK8S

	providers := map[common.Provider]bool{
		common.ProviderAWS: enableAWS || enableALL,
		common.ProviderAZ:  enableAZ || enableALL,
		common.ProviderGCP: enableGCP || enableALL,
		common.ProviderK8S: enableK8S || enableALL,
	}

	if testIdentifier == "" {
		testIdentifier = "smoke test"
	}

	for provider, enabled := range providers {
		if !enabled {
			continue
		}

		t.Run(string(provider), func(t *testing.T) {
			oldData := gofakeit.UUID()
			newData := gofakeit.UUID()

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
						mv data /tmp/data
						echo "$ENVIRONMENT_VARIABLE_DATA" | tee data
						sleep 60
						cat /tmp/data
					`,
					Variables: map[string]*string{
						"ENVIRONMENT_VARIABLE_DATA": &newData,
					},
					Directory: dataDirectory,
					DirectoryOut: dataDirectory,
					Timeout:   10 * time.Minute,
				},
				Firewall: common.Firewall{
					Ingress: common.FirewallRule{
						Ports: &[]uint16{22},
					},
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
					if strings.Contains(log, oldData) &&
						strings.Contains(log, newData) {
						break loop
					}
				}

				time.Sleep(10 * time.Second)
			}

			if provider == common.ProviderK8S {
				require.Equal(t, newTask.Start(ctx), common.NotImplementedError)
				require.Equal(t, newTask.Stop(ctx), common.NotImplementedError)
			} else {
				for assert.Nil(t, newTask.Read(ctx)) {
					status, err := newTask.Status(ctx)
					require.Nil(t, err)
					if status[common.StatusCodeActive] == 0 &&
						status[common.StatusCodeSucceeded] > 0 {
						break
					} else {
						time.Sleep(10 * time.Second)
					}
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
