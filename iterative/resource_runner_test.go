package iterative

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sebdah/goldie/v2"

	"github.com/stretchr/testify/assert"
)

func TestScript(t *testing.T) {
	t.Run("AWS known region should not add the NVIDA drivers", func(t *testing.T) {
		data := make(map[string]interface{})
		data["ami"] = isAMIAvailable("aws", "us-east-1")

		script, err := renderScript(data)
		assert.Nil(t, err)
		assert.NotContains(t, script, "sudo ubuntu-drivers autoinstall")
	})

	t.Run("AWS known generic region should not add the NVIDA drivers", func(t *testing.T) {
		data := make(map[string]interface{})
		data["ami"] = isAMIAvailable("aws", "us-west")

		script, err := renderScript(data)
		assert.Nil(t, err)
		assert.NotContains(t, script, "sudo ubuntu-drivers autoinstall")
	})

	t.Run("AWS unknown region should add the NVIDA drivers", func(t *testing.T) {
		data := make(map[string]interface{})
		data["ami"] = isAMIAvailable("aws", "us-east-99")

		script, err := renderScript(data)
		assert.Nil(t, err)
		assert.Contains(t, script, "sudo ubuntu-drivers autoinstall")
	})

	t.Run("Azure known region should add the NVIDA drivers", func(t *testing.T) {
		data := make(map[string]interface{})
		data["ami"] = isAMIAvailable("azure", "westus")

		script, err := renderScript(data)
		assert.Nil(t, err)
		assert.Contains(t, script, "sudo ubuntu-drivers autoinstall")
	})

	t.Run("Azure known generic region should add the NVIDA drivers", func(t *testing.T) {
		data := make(map[string]interface{})
		data["ami"] = isAMIAvailable("azure", "us-west")

		script, err := renderScript(data)
		assert.Nil(t, err)
		assert.Contains(t, script, "sudo ubuntu-drivers autoinstall")
	})

	t.Run("Azure unknown region should add the NVIDA drivers", func(t *testing.T) {
		data := make(map[string]interface{})
		data["ami"] = isAMIAvailable("azure", "us-east-99")

		script, err := renderScript(data)
		assert.Nil(t, err)
		assert.Contains(t, script, "sudo ubuntu-drivers autoinstall")
	})

	t.Run("Runner Startup Script", func(t *testing.T) {
		data := make(map[string]interface{})
		startupScript, _ := base64.StdEncoding.DecodeString("ZWNobyAiaGVsbG8gd29ybGQiCmVjaG8gImJ5ZSB3b3JsZCI=")
		data["runner_startup_script"] = string(startupScript)

		script, err := renderScript(data)
		assert.Nil(t, err)
		assert.Contains(t, script, "echo \"hello world\"\necho \"bye world\"")
	})
}

func TestProvisionerCode(t *testing.T) {
	g := goldie.New(t, goldie.WithDiffEngine(goldie.ColoredDiff))

	t.Run("AWS provisioner code should pass golden test", func(t *testing.T) {
		val, err := renderProvisionerCode(t, "aws")
		assert.Nil(t, err)
		g.Assert(t, "script_template_cloud_aws", []byte(val))
	})

	t.Run("Azure provisioner code should pass golden test", func(t *testing.T) {
		val, err := renderProvisionerCode(t, "azure")
		assert.Nil(t, err)
		g.Assert(t, "script_template_cloud_azure", []byte(val))
	})

	t.Run("Kubernetes provisioner code should pass golden test", func(t *testing.T) {
		val, err := renderProvisionerCode(t, "kubernetes")
		assert.Nil(t, err)
		g.Assert(t, "script_template_cloud_kubernetes", []byte(val))
	})

	t.Run("Invalid cloud provisioner code should pass golden test", func(t *testing.T) {
		val, err := renderProvisionerCode(t, "invalid")
		assert.Nil(t, err)
		g.Assert(t, "script_template_cloud_invalid", []byte(val))
	})
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
		"AWS_SECRET_ACCESS_KEY":    "1 value with \"quotes\" and spaces",
		"AWS_ACCESS_KEY_ID":        "2 value with \"quotes\" and spaces",
		"AZURE_CLIENT_ID":          "3 value with \"quotes\" and spaces",
		"AZURE_CLIENT_SECRET":      "4 value with \"quotes\" and spaces",
		"AZURE_SUBSCRIPTION_ID":    "5 value with \"quotes\" and spaces",
		"AZURE_TENANT_ID":          "6 value with \"quotes\" and spaces",
		"KUBERNETES_CONFIGURATION": "7 value with \"quotes\" and spaces",
	}
}
