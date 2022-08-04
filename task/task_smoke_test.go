//go:build smoke_tests

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

func TestTaskSmokeTest(t *testing.T) {
	testName := os.Getenv("SMOKE_TEST_IDENTIFIER")
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

	if testName == "" {
		testName = "smoke test"
	}

	for provider, enabled := range providers {
		if !enabled {
			continue
		}

		t.Run(string(provider), func(t *testing.T) {
			oldData := gofakeit.UUID()
			newData := gofakeit.UUID()

			baseDirectory := t.TempDir()
			cacheDirectory := filepath.Join(baseDirectory, "cache")
			outputDirectory := filepath.Join(baseDirectory, "output")
			cacheFile := filepath.Join(cacheDirectory, "file")
			outputFile := filepath.Join(outputDirectory, "file")

			relativeOutputDirectory, err := filepath.Rel(baseDirectory, outputDirectory)
			require.NoError(t, err)

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

			identifier := common.NewIdentifier(testName)

			task := common.Task{
				Size: common.Size{
					Machine: "m+t4",
				},
				Environment: common.Environment{
					Image: "nvidia",
					Script: `#!/bin/sh -e
						nvidia-smi
						mkdir --parents cache output
						touch cache/file
						echo "$ENVIRONMENT_VARIABLE_DATA" | tee --append output/file
						sleep 60
						cat output/file
					`,
					Variables: map[string]*string{
						"ENVIRONMENT_VARIABLE_DATA": &newData,
					},
					Directory:    baseDirectory,
					DirectoryOut: relativeOutputDirectory,
					Timeout:      10 * time.Minute,
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
			require.NoError(t, err)

			require.NoError(t, newTask.Delete(ctx))
			if sweepOnly {
				return
			}

			require.NoError(t, os.Mkdir(cacheDirectory, 0777))
			require.NoError(t, os.Mkdir(outputDirectory, 0777))

			file, err := os.Create(outputFile)
			require.NoError(t, err)

			_, err = file.WriteString(oldData)
			require.NoError(t, err)
			require.NoError(t, file.Close())

			require.NoError(t, newTask.Create(ctx))
			require.NoError(t, newTask.Create(ctx))

		loop:
			for assert.NoError(t, newTask.Read(ctx)) {
				logs, err := newTask.Logs(ctx)
				require.NoError(t, err)
				t.Log(logs)

				for _, log := range logs {
					if strings.Contains(log, oldData) &&
						strings.Contains(log, newData) {
						break loop
					}
				}

				status, err := newTask.Status(ctx)
				require.NoError(t, err)
				t.Log(status)

				if status[common.StatusCodeFailed] > 0 {
					break
				}

				time.Sleep(10 * time.Second)
			}

			if provider == common.ProviderK8S {
				require.Equal(t, newTask.Start(ctx), common.NotImplementedError)
				require.Equal(t, newTask.Stop(ctx), common.NotImplementedError)
			}

			for assert.NoError(t, newTask.Read(ctx)) {
				status, err := newTask.Status(ctx)
				require.NoError(t, err)

				if status[common.StatusCodeActive] == 0 &&
					status[common.StatusCodeSucceeded] > 0 {
					break
				}

				time.Sleep(10 * time.Second)
			}

			require.NoError(t, newTask.Delete(ctx))
			require.NoError(t, newTask.Delete(ctx))

			require.NoFileExists(t, cacheFile)
			require.FileExists(t, outputFile)

			contents, err := ioutil.ReadFile(outputFile)
			require.NoError(t, err)

			require.Contains(t, string(contents), newData)
		})
	}
}
