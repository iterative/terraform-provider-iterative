package utils

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wessie/appdirs"
)

func TestVersion(t *testing.T) {
	assert.Equal(t, Version, "0.0.0")
}

func TestTerraformVersion(t *testing.T) {
	ver := TerraformVersion()
	assert.Equal(t, strings.HasPrefix(ver, "v"), true)
}

func TestSystemInfo(t *testing.T) {
	info := SystemInfo()
	assert.NotNil(t, info["os_name"])
	assert.NotNil(t, info["platform_version"])
}

func TestUserId(t *testing.T) {
	old := appdirs.UserConfigDir("dvc/user_id", "iterative", "", false)
	new := appdirs.UserConfigDir("iterative/telemetry", "", "", false)

	userId := "00000000-0000-0000-0000-000000000000"
	data := map[string]interface{}{
		"user_id": userId,
	}
	json, _ := json.MarshalIndent(data, "", " ")

	os.MkdirAll(filepath.Dir(old), 0644)
	_ = ioutil.WriteFile(old, json, 0644)

	id := UserId()
	assert.Equal(t, len(id) == 36, true)

	if !IsCI() {
		assert.Equal(t, userId == id, true)

		_, err := os.Stat(new)
		assert.Equal(t, !os.IsNotExist(err), true)
	}
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
