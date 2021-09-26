package iterative

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
)

func Test_renderDVCScript(t *testing.T) {
	type args struct {
		data map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"test startup script",
			args{
				map[string]interface{}{
					"startup_script": "ZWNobyAiaGVsbG8gd29ybGQiCmVjaG8gImJ5ZSB3b3JsZCI=",
				},
			},
			"echo \"hello world\"\necho \"bye world\"", // contains
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderDVCScript(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderDVCScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("renderDVCScript() = %v, not contains %v", got, tt.want)
			}
		})
	}
}

func Test_dvcRunProvisioner(t *testing.T) {
	g := goldie.New(t, goldie.WithDiffEngine(goldie.ColoredDiff))

	for _, cloud := range []string{"aws", "azure", "gcp", "kubernetes"} {
		t.Run("Provisioner code for "+cloud+" should pass golden test", func(t *testing.T) {
			val, err := renderProvisionerCodeForDVC(t, cloud)
			assert.Nil(t, err)
			g.Assert(t, "script_template_dvc_cloud_"+cloud, []byte(val))
		})
	}
}

func renderProvisionerCodeForDVC(t *testing.T, cloud string) (string, error) {
	// Note for future tests: this code modifies process environment variables.
	for key, value := range generateEnvironmentTestData(t) {
		os.Setenv(key, value)
	}
	return dvcRunProvisioner(generateSchemaTestDataForDVC(cloud, t))
}

func generateSchemaTestDataForDVC(cloud string, t *testing.T) *schema.ResourceData {
	return schema.TestResourceDataRaw(t, resourceDVCRun().Schema, map[string]interface{}{
		"cloud":                 cloud,
		"region":                "9 value with \"quotes\" and spaces",
		"name":                  "10 value with \"quotes\" and spaces",
		"single":                true,
		"idle_timeout":          11,
		"instance_hdd_size":     12,
		"dvc_ver":               "13 value with \"quotes\" and spaces",
		"repo":                  "14 value with \"quotes\" and spaces",
		"labels":                "15 value with \"quotes\" and spaces",
		"instance_gpu":          "16 value with \"quotes\" and spaces",
		"runner_startup_script": "echo \"custom startup script\"",
	})
}
