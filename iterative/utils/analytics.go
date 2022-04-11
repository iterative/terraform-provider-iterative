package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/shirou/gopsutil/host"
	"github.com/sirupsen/logrus"
)

func SystemInfo() map[string]interface{} {
	hostStat, _ := host.Info()
	return map[string]interface{}{
		"os":               hostStat.OS,
		"platform":         hostStat.Platform,
		"platform_version": hostStat.PlatformVersion,
	}
}

func TaskDuration(logs string) float64 {
	regex := regexp.MustCompile(`\w{3} \d{2} \d{2}:\d{2}:\d{2}`)
	matches := regex.FindAllString(logs, -1)
	taskDuration := 0.0

	if len(matches) > 1 {
		layout := "Mar 07 06:43:27"
		t1, _ := time.Parse(layout, matches[len(matches)-1])
		fmt.Println(t1)
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

func version() string {
	return "v1.0.0"
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

func JitsuUserId() string {
	hostStat, _ := host.Info()
	id := hostStat.HostID()

	if IsCI() {
		// case GitHub:
		//     group_id = "$GITHUB_SERVER_URL/$(dirname "$GITHUB_REPOSITORY")"
		//     user_id = "$GITHUB_ACTOR" # only the user name
		// case GitLab:
		//     group_id = "$CI_SERVER_URL/$CI_PROJECT_ROOT_NAMESPACE"
		//     user_id = "$CI_COMMIT_AUTHOR" # equivalent to git log below
		// case Bitbucket:
		//     group_id "$BITBUCKET_WORKSPACE"
		//     user_id = "$(git log -1 --pretty=format:'%an %ae')"
		// *_id = uuid5(sha256(*_id)[:8])

		//id = fmt.Sprintf("%s%s%s%s%s%s", os.Getenv("GITHUB_SERVER_URL"))
	}

	space := uuid.MustParse("c62985c8-2e8c-45fa-93af-4b9f577ed49e")
	hash := md5.Sum([]byte(id))
	return uuid.NewSHA1(space, hash[:8])
}

func JitsuResourceData(d *schema.ResourceData) map[string]interface{} {
	if d != nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"cloud":     d.Get("cloud").(string),
		"region":    d.Get("region").(string),
		"machine":   d.Get("machine").(string),
		"disk_size": d.Get("disk_size").(int),
		"spot":      d.Get("spot").(float64),
		"status":    d.Get("status").(map[string]interface{}),
		"duration":  TaskDuration(d.Get("logs").(string)),
	}
}

func JitsuEventPayload(eventType string, eventName string, d *schema.ResourceData) map[string]interface{} {
	payload := map[string]interface{}{
		"event_type":    eventType,
		"event_name":    eventName,
		"version":       version(),
		"tf_version":    TerraformVersion(),
		"user_id":       JitsuUserId(),
		"system_info":   SystemInfo(),
		"ci":            IsCI(),
		"resource_data": JitsuResourceData(d),
	}

	return payload
}

func SendJitsuEvent(eventType string, eventName string, d *schema.ResourceData) {
	if _, ok := os.LookupEnv("DO_NOT_TRACK"); ok {
		logrus.Debug("Analytics: DO_NOT_TRACK enabled")
		return
	}

	postBody, _ := json.Marshal(JitsuEventPayload(eventType, eventName, d))

	host := "https://0f82a78ac1e6.ngrok.io"
	token := "s2s.vy0gtflakci9wbwyootfqy.7ztlcr98n9l02b55kktmncg"
	url := host + "/api/v1/s2s/event?ip_policy=strict&token=" + token
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		logrus.Error("Analytics: failed sending event: %v", err)
	}
	defer resp.Body.Close()
}
