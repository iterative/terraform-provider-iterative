package utils

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v45/github"
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

func deterministic(data string) (string, error) {
	ns := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("iterative.ai"))

	seed, err := ns.MarshalBinary()
	if err != nil {
		return "", err
	}

	dk, err := scrypt.Key([]byte(data), seed, 1<<16, 8, 1, 8)
	if err != nil {
		return "", err
	}

	id := uuid.NewSHA1(ns, []byte(hex.EncodeToString(dk)))
	return id.String(), nil
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
	return len(guessCI()) > 0
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

	if _, ok := os.LookupEnv("CI"); ok {
		return "unknown"
	}

	return ""
}

func TerraformVersion() (string, error) {
	var out bytes.Buffer
	cmd := exec.Command("terraform", "--version")
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	regex := regexp.MustCompile(`v\d+\.\d+\.\d+`)
	version := regex.FindString(out.String())
	return version, nil
}

func GroupId() (string, error) {
	if !IsCI() {
		return "", nil
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
		rawId = fmt.Sprintf("https://bitbucket.com/%s",
			os.Getenv("BITBUCKET_WORKSPACE"))
	}

	id, err := deterministic(rawId)
	if err != nil {
		return "", err
	}

	return id, nil
}

func readId(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	var data map[string]interface{}

	if err := json.Unmarshal([]byte(bytes), &data); err != nil {
		uid, uidError := uuid.FromBytes(bytes)
		if uidError != nil {
			return "", fmt.Errorf("failed parsing user_id as json and plaintext: %w", uidError)
		}
		logrus.Traceln(fmt.Errorf("found old format telemtry uid, json err: %w", err))
		return uid.String(), nil
	}

	if id, ok := data["user_id"].(string); ok {
		return id, nil
	}

	return "", errors.New("user_id not found or not a string")
}

func writeId(path string, id string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data := map[string]string{"user_id": id}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, bytes, 0644)
}

func UserId() (string, error) {
	var id string
	var err error
	if IsCI() {
		ci := guessCI()
		var rawId string

		if ci == "github" {
			client := github.NewClient(nil)
			user, _, err := client.Users.Get(context.Background(), os.Getenv("GITHUB_ACTOR"))
			if err != nil {
				return "", err
			}
			name := ""
			if user.Name != nil {
				name = *user.Name
			}

			rawId = fmt.Sprintf("%s %s %d", name, *user.Login, *user.ID)
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
				return "", err
			}

			rawId = out.String()
		}

		id, err := deterministic(rawId)
		if err != nil {
			return "", err
		}

		return id, nil
	}

	id = uuid.New().String()
	old := appdirs.UserConfigDir("dvc/user_id", "iterative", "", false)
	_, errorOld := os.Stat(old)

	new := appdirs.UserConfigDir("iterative/telemetry", "", "", false)
	_, errorNew := os.Stat(new)

	if errors.Is(errorNew, fs.ErrNotExist) {
		if !errors.Is(errorOld, fs.ErrNotExist) {
			id, err = readId(old)
			if err != nil {
				return "", err
			}
		}

		err = writeId(new, id)
		if err != nil {
			return "", err
		}
	} else {
		id, err = readId(new)
		if err != nil {
			return "", err
		}
	}

	if os.IsNotExist(errorOld) && id != "do-not-track" {
		err := writeId(old, id)
		if err != nil {
			return "", err
		}
	}

	return id, nil
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

func JitsuEventPayload(action string, e error, extra map[string]interface{}) (map[string]interface{}, error) {
	systemInfo := SystemInfo()

	userId, uErr := UserId()
	if uErr != nil {
		return nil, uErr
	}

	groupId, gErr := GroupId()
	if gErr != nil {
		return nil, gErr
	}

	tfVer, tfVerErr := TerraformVersion()
	if tfVerErr != nil {
		return nil, tfVerErr
	}

	extra["ci"] = guessCI()
	extra["terraform_version"] = tfVer

	payload := map[string]interface{}{
		"user_id":      userId,
		"group_id":     groupId,
		"action":       action,
		"interface":    "cli",
		"tool_name":    "tpi",
		"tool_source":  "terraform",
		"tool_version": Version,
		"os_name":      systemInfo["os_name"],
		"os_version":   systemInfo["platform_version"],
		"backend":      extra["cloud"],
		"extra":        extra,
	}

	if e != nil { 
		payload["error"] = reflect.TypeOf(e).String()
	}

	return payload, nil
}

func SendJitsuEvent(action string, e error, extra map[string]interface{}) {
	for _, env := range []string{"ITERATIVE_DO_NOT_TRACK"} {
		if _, ok := os.LookupEnv(env); ok {
			logrus.Debugf("analytics: %s environment variable is set; doing nothing", env)
			return
		}
	}

	// Exclude runs from GitHub Codespaces at Iterative
	if strings.HasPrefix(os.Getenv("GITHUB_REPOSITORY"), "iterative/") {
		return
	}

	payload, err := JitsuEventPayload(action, e, extra)
	if err != nil {
		logrus.Debugf("analytics: Failure generating Jitsu Event Payload; doing nothing")
		return
	}

	// Exclude continuous integration tests and internal projects from analytics
	for _, group := range []string{
		"dc16cd76-71b7-5afa-bf11-e85e02ee1554", // deterministic("https://github.com/iterative")
		"b0e229bf-2598-54b7-a3e0-81869cdad579", // deterministic("https://github.com/iterative-test")
		"d5aaeca4-fe6a-5c72-8aa7-6dcd65974973", // deterministic("https://gitlab.com/iterative.ai")
		"b6df227b-5b3d-5190-a8fa-d272b617ee6c", // deterministic("https://gitlab.com/iterative-test")
		"2c6415f0-cb5a-5e52-8c81-c5af4f11715d", // deterministic("https://bitbucket.com/iterative-ai")
		"c0b86b90-d63c-5fb0-b84d-718d8e15f8d6", // deterministic("https://bitbucket.com/iterative-test")
	} {
		if payload["group_id"].(string) == group {
			return
		}
	}

	if payload["user_id"] == "do-not-track" {
		logrus.Debugf("analytics: user_id %s is set; doing nothing", payload["user_id"])
		return
	}

	go send(payload) //nolint:errcheck
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
