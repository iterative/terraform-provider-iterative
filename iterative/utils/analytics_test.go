package utils

import (
	"strings"
	"testing"

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
	assert.Equal(t, 120.0, data["task_duration"].(float64))
	assert.Equal(t, true, data["task_resumed"].(bool))
}
