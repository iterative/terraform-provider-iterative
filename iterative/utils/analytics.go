package utils

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
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

	id, err := deterministic(fmt.Sprintf("%s%s%s%s%s",
		os.Getenv("GITHUB_SERVER_URL"),
		os.Getenv("GITHUB_REPOSITORY_OWNER"),
		os.Getenv("CI_SERVER_URL"),
		os.Getenv("CI_PROJECT_ROOT_NAMESPACE"),
		os.Getenv("BITBUCKET_WORKSPACE"),
	))

	if err != nil {
		return ""
	}

	return id.String()
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
		uid, err := deterministic(fmt.Sprintf("%s%s%s",
			os.Getenv("GITHUB_ACTOR"),
			os.Getenv("GITLAB_USER_ID"),
			os.Getenv("BITBUCKET_STEP_TRIGGERER_UUID"),
		))

		if err == nil {
			id = uid.String()
		}
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
		err = reflect.TypeOf(e).Name()
	}

	payload := map[string]interface{}{
		"user_id":      UserId(),
		"group_id":     GroupId(),
		"action":       action,
		"interface":    "cli",
		"tool_name":    "tpi",
		"tool_source":  "terraform",
		"tool_version": Version,
		"os_name":      systemInfo["platform"],
		"os_version":   systemInfo["platform_version"],
		"backend":      extra["cloud"],
		"error":        err,
		"extra":        extra,
	}

	return payload
}

func SendJitsuEvent(ctx context.Context, action string, e error, extra map[string]interface{}) {
	for _, prefix := range []string{"ITERATIVE", "DVC"} {
		if _, ok := os.LookupEnv(prefix + "_NO_ANALYTICS"); ok {
			logrus.Debugf("analytics: %s_NO_ANALYTICS environment variable is set", prefix)
			return
		}
	}

	wg.Add(1)
	ctx, _ = context.WithTimeout(ctx, Timeout)
	go send(ctx, JitsuEventPayload(action, e, extra))
}

func send(ctx context.Context, event interface{}) error {
	defer wg.Done()

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", getenv("TPI_ANALYTICS_ENDPOINT", Endpoint), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", getenv("TPI_ANALYTICS_TOKEN", Token))

	resp, err := http.DefaultClient.Do(req)
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
		SendJitsuEvent(context.Background(), "panic", fmt.Errorf("panic: %v", r), extra)
	}

	wg.Wait()

	if r != nil {
		panic(r)
	}
}
