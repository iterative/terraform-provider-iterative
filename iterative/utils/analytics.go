package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/shirou/gopsutil/host"
	"github.com/sirupsen/logrus"
	"github.com/wessie/appdirs"
)

var (
	Version string = "0.0.0"
)

func getenv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

func idgen(data string) string {
	space := uuid.MustParse("c62985c8-2e8c-45fa-93af-4b9f577ed49e")
	hash := md5.Sum([]byte(data))
	uuid := uuid.NewSHA1(space, hash[:8]).String()
	return uuid
}

func SystemInfo() map[string]interface{} {
	hostStat, _ := host.Info()
	return map[string]interface{}{
		"platform":         hostStat.Platform,
		"platform_version": hostStat.PlatformVersion,
	}
}

func TaskDuration(logs string) float64 {
	regex := regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)
	matches := regex.FindAllString(logs, -1)
	taskDuration := 0.0

	if len(matches) > 1 {
		layout := "2006-01-02 15:04:05"
		t1, _ := time.Parse(layout, matches[len(matches)-1])
		t0, _ := time.Parse(layout, matches[0])
		taskDuration = t1.Sub(t0).Seconds()
	}

	return taskDuration
}

func IsCI() bool {
	if _, ok := os.LookupEnv("CI"); ok {
		return true
	}

	return false
}

func TerraformVersion() string {
	var out bytes.Buffer
	cmd := exec.Command("terraform", "--version")
	cmd.Stdout = &out

	err := cmd.Run()

	if err == nil {
		regex := regexp.MustCompile(`v\d+\.\d+\.\d+`)
		version := regex.FindString(out.String())
		return version
	} else {
		logrus.Error("Analytics: Failed extracting terraform version")
	}

	return ""
}

func GroupId() string {
	if !IsCI() {
		return ""
	}

	id := idgen(fmt.Sprintf("%s%s%s%s%s",
		os.Getenv("GITHUB_SERVER_URL"),
		os.Getenv("GITHUB_REPOSITORY_OWNER"),
		os.Getenv("CI_SERVER_URL"),
		os.Getenv("CI_PROJECT_ROOT_NAMESPACE"),
		os.Getenv("BITBUCKET_WORKSPACE"),
	))

	return id
}

func UserId() string {
	id := uuid.New().String()
	old := appdirs.UserConfigDir("dvc/user_id", "iterative", "", false)
	new := appdirs.UserConfigDir("telemetry", "iterative", "", false)

	_, errorOld := os.Stat(old)
	if !os.IsNotExist(errorOld) {
		os.Rename(old, new)
	}

	_, errorNew := os.Stat(new)
	if !os.IsNotExist(errorNew) {
		dat, _ := ioutil.ReadFile(new)
		id = string(dat[:])
	} else {
		ioutil.WriteFile(new, []byte(id), 0644)
	}

	if IsCI() {
		id = idgen(fmt.Sprintf("%s%s%s",
			os.Getenv("GITHUB_ACTOR"),
			os.Getenv("GITLAB_USER_ID"),
			os.Getenv("BITBUCKET_STEP_TRIGGERER_UUID"),
		))
	}

	return id
}

func ResourceData(d *schema.ResourceData) map[string]interface{} {
	if d == nil {
		return map[string]interface{}{}
	}

	tpiLogs := d.Get("logs").([]interface{})
	logs := ""
	for _, log := range tpiLogs {
		logs += log.(string)
	}
	spot := d.Get("spot").(float64)
	return map[string]interface{}{
		"cloud":           d.Get("cloud").(string),
		"cloud_region":    d.Get("region").(string),
		"cloud_machine":   d.Get("machine").(string),
		"cloud_disk_size": d.Get("disk_size").(int),
		"cloud_spot":      spot,
		"cloud_spot_auto": spot == 0.0,
		"task_status":     d.Get("status").(map[string]interface{}),
		"task_duration":   TaskDuration(logs),
		"task_resumed":    len(logs) > 1,
	}
}

func JitsuEventPayload(action string, e error, d *schema.ResourceData) map[string]interface{} {
	systemInfo := SystemInfo()

	extra := ResourceData(d)
	extra["ci"] = IsCI()
	extra["terraform_version"] = TerraformVersion()

	err := ""
	if e != nil {
		err = e.Error()
	}

	payload := map[string]interface{}{
		"user_id":      UserId(),
		"group_id":     GroupId(),
		"action":       action,
		"interface":    "cli",
		"tool_name":    "tpi",
		"tool_source":  "terraform",
		"tool_version": Version,
		"os_name":      systemInfo["plattform"],
		"os_version":   systemInfo["platform_version"],
		"backend":      extra["cloud"],
		"error":        err,
		"extra":        extra,
	}

	return payload
}

func SendJitsuEvent(action string, e error, d *schema.ResourceData) {
	if d == nil {
		return
	}

	if _, ok := os.LookupEnv("DO_NOT_TRACK"); ok {
		logrus.Debug("Analytics: DO_NOT_TRACK enabled")
		return
	}

	postBody, _ := json.Marshal(JitsuEventPayload(action, e, d))

	host := getenv("TPI_ANALYTICS_HOST", "https://telemetry.cml.dev")
	token := getenv("TPI_ANALYTICS_KEY", "s2s.jtyjusrpsww4k9b76rrjri.bl62fbzrb7nd9n6vn5bpqt")
	url := host + "/api/v1/s2s/event?ip_policy=strict&token=" + token
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		logrus.Error("Analytics: failed sending event")
	}
	defer resp.Body.Close()
}
