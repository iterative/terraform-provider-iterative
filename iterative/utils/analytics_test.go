package utils

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	assert.Equal(t, strings.HasPrefix(Version, "v"), true)
}

func TestTerraformVersion(t *testing.T) {
	ver := TerraformVersion()
	assert.Equal(t, strings.HasPrefix(ver, "v"), true)
}

func TestSystemInfo(t *testing.T) {
	info := SystemInfo()
	assert.NotNil(t, info["platform"])
	assert.NotNil(t, info["platform_version"])
}

func TestUserId(t *testing.T) {
	id := UserId()
	assert.Equal(t, len(id) == 36, true)
}

func TestResourceData(t *testing.T) {
	cloud := "aws"
	region := "us-west"
	machine := "xl"
	disk_size := 30
	spot := 0.0
	status := map[string]interface{}{
		"running": 0,
		"failed":  0,
	}
	logs := make([]interface{}, 0)
	logs = append(logs, "2022-04-24 20:25:07 Started tpi-task.service.\n2022-04-24 20:26:07 hello.\n")
	logs = append(logs, "2022-04-24 20:27:07 Started tpi-task.service.\n")

	d := generateSchemaData(t, map[string]interface{}{
		"cloud":     cloud,
		"region":    region,
		"machine":   machine,
		"disk_size": disk_size,
		"spot":      spot,
		"status":    status,
		"logs":      logs,
	})

	data := ResourceData(d)

	assert.Equal(t, cloud, data["cloud"].(string))
	assert.Equal(t, region, data["cloud_region"].(string))
	assert.Equal(t, machine, data["cloud_machine"].(string))
	assert.Equal(t, disk_size, data["cloud_disk_size"].(int))
	assert.Equal(t, spot, data["cloud_spot"].(float64))
	assert.Equal(t, true, data["cloud_spot_auto"].(bool))
	assert.Equal(t, status, data["task_status"].(map[string]interface{}))
	assert.Equal(t, true, data["task_duration"].(float64) > 0.0)
	assert.Equal(t, true, data["task_resumed"].(bool))
}

func generateSchemaData(t *testing.T, raw map[string]interface{}) *schema.ResourceData {
	sch := map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
		},
		"cloud": {
			Type:     schema.TypeString,
			ForceNew: true,
			Required: true,
		},
		"region": {
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
			Default:  "us-west",
		},
		"machine": {
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
			Default:  "m",
		},
		"disk_size": {
			Type:     schema.TypeInt,
			ForceNew: true,
			Optional: true,
			Default:  30,
		},
		"spot": {
			Type:     schema.TypeFloat,
			ForceNew: true,
			Optional: true,
			Default:  -1,
		},
		"image": {
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
			Default:  "ubuntu",
		},
		"addresses": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"status": {
			Type:     schema.TypeMap,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeInt,
			},
		},
		"events": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"logs": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"script": {
			Type:     schema.TypeString,
			ForceNew: true,
			Required: true,
		},
		"workdir": {
			Optional: true,
			Type:     schema.TypeSet,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"input": {
						Type:     schema.TypeString,
						ForceNew: true,
						Optional: true,
						Default:  "",
					},
					"output": {
						Type:     schema.TypeString,
						ForceNew: false,
						Optional: true,
						Default:  "",
					},
				},
			},
		},
	}

	return schema.TestResourceDataRaw(t, sch, raw)
}
