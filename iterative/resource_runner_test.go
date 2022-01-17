package iterative

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sebdah/goldie/v2"

	"github.com/stretchr/testify/assert"
)

func TestScript(t *testing.T) {
	t.Run("Runner Startup Script", func(t *testing.T) {
		data := make(map[string]interface{})
		data["startup_script"] = string("ZWNobyAiaGVsbG8gd29ybGQiCmVjaG8gImJ5ZSB3b3JsZCI=")

		script, err := renderScript(data)
		assert.Nil(t, err)
		assert.Contains(t, script, "echo \"hello world\"\necho \"bye world\"")
	})
}

func TestDockerVolumes(t *testing.T) {
	t.Run("Runner Docker Volumes", func(t *testing.T) {
		data := make(map[string]interface{})
		data["docker_volumes"] = []string{"/one/one.txt:/one/one.txt", "/two:/two"}

		script, err := renderScript(data)
		assert.Nil(t, err)
		assert.Contains(t, script, "--docker-volumes /one/one.txt:/one/one.txt --docker-volumes /two:/two")
	})
}

func TestProvisionerCode(t *testing.T) {
	g := goldie.New(t, goldie.WithDiffEngine(goldie.ColoredDiff))

	for _, cloud := range []string{"aws", "azure", "gcp", "kubernetes", "invalid"} {
		t.Run("Provisioner code for "+cloud+" should pass golden test", func(t *testing.T) {
			val, err := renderProvisionerCode(t, cloud)
			assert.Nil(t, err)
			g.Assert(t, "script_template_cloud_"+cloud, []byte(val))
		})
	}
}

func renderProvisionerCode(t *testing.T, cloud string) (string, error) {
	// Note for future tests: this code modifies process environment variables.
	for key, value := range generateEnvironmentTestData(t) {
		os.Setenv(key, value)
	}
	return provisionerCode(generateSchemaTestData(cloud, t))
}

func generateSchemaTestData(cloud string, t *testing.T) *schema.ResourceData {
	return schema.TestResourceDataRaw(t, resourceRunner().Schema, map[string]interface{}{
		"cloud":                 cloud,
		"region":                "9 value with \"quotes\" and spaces",
		"name":                  "10 value with \"quotes\" and spaces",
		"single":                true,
		"idle_timeout":          11,
		"instance_hdd_size":     12,
		"token":                 "13 value with \"quotes\" and spaces",
		"repo":                  "14 value with \"quotes\" and spaces",
		"driver":                "15 value with \"quotes\" and spaces",
		"labels":                "16 value with \"quotes\" and spaces",
		"instance_gpu":          "17 value with \"quotes\" and spaces",
		"runner_startup_script": "echo \"custom startup script\"",
	})
}

func generateEnvironmentTestData(t *testing.T) map[string]string {
	return map[string]string{
		"AWS_SECRET_ACCESS_KEY":               "0 value with \"quotes\" and spaces",
		"AWS_ACCESS_KEY_ID":                   "1 value with \"quotes\" and spaces",
		"AWS_SESSION_TOKEN":                   "2 value with \"quotes\" and spaces",
		"AZURE_CLIENT_ID":                     "3 value with \"quotes\" and spaces",
		"AZURE_CLIENT_SECRET":                 "4 value with \"quotes\" and spaces",
		"AZURE_SUBSCRIPTION_ID":               "5 value with \"quotes\" and spaces",
		"AZURE_TENANT_ID":                     "6 value with \"quotes\" and spaces",
		"GOOGLE_APPLICATION_CREDENTIALS_DATA": "7 value with \"quotes\" and spaces",
		"KUBERNETES_CONFIGURATION":            "8 value with \"quotes\" and spaces",
	}
}
