//go:build smoke

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

type testTarget struct {
	machine string
	image   string
	env     map[string]string
	spot    common.Spot
}

var testTargets = map[string]testTarget{
	// default test target spins up small instances and runs lightweight scripts on them.
	"default": testTarget{
		machine: "m",
		image:   "ubuntu",
		spot:    common.SpotDisabled,
	},
	"gpu": testTarget{
		machine: "m+t4",
		image:   "nvidia",
		env: map[string]string{
			"TEST_GPU": "yes",
		},
		spot: common.SpotEnabled,
	},
}

// TestTaskSmoke runs smoke tests with specified infrastructure providers.
// Cloud provider access credentials (provided as environment variables) are required.
func TestTaskSmoke(t *testing.T) {
	testName := os.Getenv("SMOKE_TEST_IDENTIFIER")
	sweepOnly := os.Getenv("SMOKE_TEST_SWEEP") != ""
	enableAWS := os.Getenv("SMOKE_TEST_ENABLE_AWS") != ""
	enableAZ := os.Getenv("SMOKE_TEST_ENABLE_AZ") != ""
	enableGCP := os.Getenv("SMOKE_TEST_ENABLE_GCP") != ""
	enableK8S := os.Getenv("SMOKE_TEST_ENABLE_K8S") != ""

	enableALL := !enableAWS && !enableAZ && !enableGCP && !enableK8S

	targetName := os.Getenv("SMOKE_TEST_TARGET")
	if targetName == "" {
		targetName = "default"
	}
	target, ok := testTargets[targetName]
	if !ok {
		t.Fatalf("test target %q undefined", targetName)
	}

	providers := map[common.Provider]bool{
		common.ProviderAWS: enableAWS || enableALL,
		common.ProviderAZ:  enableAZ || enableALL,
		common.ProviderGCP: enableGCP || enableALL,
		common.ProviderK8S: enableK8S || enableALL,
	}

	if testName == "" {
		testName = "smoke test"
	}
	script := `
#!/bin/sh -e
if [ -n "$TEST_GPU" ]; then
  nvidia-smi
fi
mkdir --parents cache output
touch cache/file
echo "$ENVIRONMENT_VARIABLE_DATA" | tee --append output/file
sleep 60
cat output/file
`[1:]
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

			identifier := common.NewDeterministicIdentifier(testName)

			env := map[string]*string{
				"ENVIRONMENT_VARIABLE_DATA": &newData,
			}
			for k := range target.env {
				val := target.env[k]
				env[k] = &val
			}
			task := common.Task{
				Size: common.Size{
					Machine: target.machine,
				},
				Environment: common.Environment{
					Image:        target.image,
					Script:       script,
					Variables:    env,
					Directory:    baseDirectory,
					DirectoryOut: relativeOutputDirectory,
					Timeout:      10 * time.Minute,
				},
				Firewall: common.Firewall{
					Ingress: common.FirewallRule{
						Ports: &[]uint16{22},
					},
				},
				Spot:        target.spot,
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
