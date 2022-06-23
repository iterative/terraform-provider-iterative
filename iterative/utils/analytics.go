package utils

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/shirou/gopsutil/host"
	"github.com/sirupsen/logrus"
	"github.com/wessie/appdirs"

	"golang.org/x/crypto/scrypt"
)

const (
	Timeout  = 5 * time.Second
	Endpoint = "https://telemetry.cml.dev/api/v1/s2s/event?ip_policy=strict"
	Token    = "s2s.jtyjusrpsww4k9b76rrjri.bl62fbzrb7nd9n6vn5bpqt"
)

var (
	Version string = "0.0.0"
	wg      sync.WaitGroup
)

func getenv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

func deterministic(data string) (*uuid.UUID, error) {
	ns := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("iterative.ai"))

	seed, err := ns.MarshalBinary()
	if err != nil {
		return nil, err
	}

	dk, err := scrypt.Key([]byte(data), seed, 1<<16, 8, 1, 8)
	if err != nil {
		return nil, err
	}

	id := uuid.NewSHA1(ns, []byte(hex.EncodeToString(dk)))
	return &id, nil
}

func SystemInfo() map[string]interface{} {
	hostStat, _ := host.Info()
	return map[string]interface{}{
		"os_name":          hostStat.OS,
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
    _, ciIsSet := os.LookupEnv("CI")
    _, tfBuildIsSet := os.LookupEnv("TF_BUILD")
    if ciIsSet || tfBuildIsSet {
		return true
	}

	if len(guessCI()) > 0 {
		return true
	}

	return false
}

func guessCI() string {
	if _, ok := os.LookupEnv("GITHUB_SERVER_URL"); ok {
		return "github"
	}

	if _, ok := os.LookupEnv("CI_SERVER_URL"); ok {
		return "gitlab"
	}

	if _, ok := os.LookupEnv("BITBUCKET_WORKSPACE"); ok {
		return "bitbucket"
	}

	if _, ok := os.LookupEnv("TF_BUILD"); ok {
		return "azure"
	}

	return ""
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

	rawId := "CI"
	ci := guessCI()

	if ci == "github" {
		rawId = fmt.Sprintf("%s/%s",
			os.Getenv("GITHUB_SERVER_URL"),
			os.Getenv("GITHUB_REPOSITORY_OWNER"))
	} else if ci == "gitlab" {
		rawId = fmt.Sprintf("%s/%s",
			os.Getenv("CI_SERVER_URL"),
			os.Getenv("CI_PROJECT_ROOT_NAMESPACE"))
	} else if ci == "bitbucket" {
		rawId = os.Getenv("BITBUCKET_WORKSPACE")
	}

	id, err := deterministic(rawId)
	if err != nil {
		return ""
	}

	return id.String()
}

func UserId() string {
	if IsCI() {
		ci := guessCI()
		var rawId string

		if ci == "github" {
			rawId = os.Getenv("GITHUB_ACTOR")
		} else if ci == "gitlab" {
			rawId = fmt.Sprintf("%s %s %s",
				os.Getenv("GITLAB_USER_NAME"),
				os.Getenv("GITLAB_USER_LOGIN"),
				os.Getenv("GITLAB_USER_ID"))
		} else if ci == "bitbucket" {
			rawId = os.Getenv("BITBUCKET_STEP_TRIGGERER_UUID")
		} else {
			var out bytes.Buffer
			cmd := exec.Command("git", "log", "-1", "--pretty=format:'%ae'")
			cmd.Stdout = &out

			err := cmd.Run()
			if err != nil {
				return ""
			}

			rawId = out.String()
		}

		id, err := deterministic(rawId)
		if err != nil {
			return ""
		}

		return id.String()
	}

	id := uuid.New().String()
	old := appdirs.UserConfigDir("dvc/user_id", "iterative", "", false)
	_, errorOld := os.Stat(old)

	new := appdirs.UserConfigDir("iterative/telemetry", "", "", false)
	_, errorNew := os.Stat(new)

	if os.IsNotExist(errorNew) {
		if !os.IsNotExist(errorOld) {
			jsonFile, jsonErr := os.Open(old)

			if jsonErr == nil {
				byteValue, _ := ioutil.ReadAll(jsonFile)
				var data map[string]interface{}
				json.Unmarshal([]byte(byteValue), &data)
				id = data["user_id"].(string)

				defer jsonFile.Close()
			}
		}

		os.MkdirAll(filepath.Dir(new), 0644)
		ioutil.WriteFile(new, []byte(id), 0644)
	} else {
		dat, _ := ioutil.ReadFile(new)
		id = string(dat[:])
	}

	if os.IsNotExist(errorOld) {
		os.MkdirAll(filepath.Dir(old), 0644)
		data := map[string]interface{}{
			"user_id": id,
		}
		file, _ := json.MarshalIndent(data, "", " ")
		ioutil.WriteFile(old, file, 0644)
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
		"task_resumed":    len(tpiLogs) > 1,
	}
}

func JitsuEventPayload(action string, e error, extra map[string]interface{}) map[string]interface{} {
	systemInfo := SystemInfo()

	extra["ci"] = guessCI()
	extra["terraform_version"] = TerraformVersion()

	err := ""
	if e != nil {
		err = reflect.TypeOf(e).String()
	}

	payload := map[string]interface{}{
		"user_id":      UserId(),
		"group_id":     GroupId(),
		"action":       action,
		"interface":    "cli",
		"tool_name":    "tpi",
		"tool_source":  "terraform",
		"tool_version": Version,
		"os_name":      systemInfo["os_name"],
		"os_version":   systemInfo["platform_version"],
		"backend":      extra["cloud"],
		"error":        err,
		"extra":        extra,
	}

	return payload
}

func SendJitsuEvent(action string, e error, extra map[string]interface{}) {
	for _, env := range []string{"ITERATIVE_DO_NOT_TRACK"} {
		if _, ok := os.LookupEnv(env); ok {
			logrus.Debugf("analytics: %s environment variable is set; doing nothing", env)
			return
		}
	}

	go send(JitsuEventPayload(action, e, extra))
	wg.Add(1)
}

func send(event interface{}) error {
	defer wg.Done()

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, getenv("TPI_ANALYTICS_ENDPOINT", Endpoint), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", getenv("TPI_ANALYTICS_TOKEN", Token))

	client := &http.Client{Timeout: Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("server returned: " + resp.Status)
	}

	return nil
}

func WaitForAnalyticsAndHandlePanics() {
	r := recover()

	if r != nil {
		extra := map[string]interface{}{"stack": debug.Stack()}
		SendJitsuEvent("panic", fmt.Errorf("panic: %v", r), extra)
	}

	wg.Wait()

	if r != nil {
		panic(r)
	}
}
