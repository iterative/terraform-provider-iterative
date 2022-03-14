package utils

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sirupsen/logrus"
)

var baseTimestamp = time.Now()
var colors = make(map[string]int)

type basicFormatter struct{}

func (f *basicFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	levelText := strings.ToUpper(entry.Level.String())
	levelColor := colors[levelText]
	tpl := "[%s] 🚀\x1b[%dmTPI\x1b[0m %s\n"
	return []byte(fmt.Sprintf(tpl, levelText, levelColor, entry.Message)), nil
}

func init() {
	colors["DEBUG"] = 34
	colors["INFO"] = 36
	colors["WARN"] = 33
	colors["ERROR"] = 31
	colors["FATAL"] = 31
	colors["foreground"] = 35

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&basicFormatter{})
}

type tpiFormatter struct{}

func (f *tpiFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(logrus.Fields)
	for k, v := range entry.Data {
		data[k] = v
	}

	if data["d"] == nil {
		return nil, errors.New("ResourceData is not yet available")
	}

	d := data["d"].(*schema.ResourceData)
	message := entry.Message
	levelText := strings.ToUpper(entry.Level.String())
	levelColor := colors[levelText]

	if message == "instance" {
		cloud := d.Get("cloud").(string)
		machine := d.Get("machine").(string)
		region := d.Get("region").(string)
		spot := d.Get("spot").(float64)

		spottext := ""
		if spot > 0 {
			spottext = fmt.Sprintf("(Spot %f/h)", spot)
		}
		message = fmt.Sprintf("%s %s%s in %s", cloud, machine, spottext, region)
	}

	if message == "status" {
		status := d.Get("status").(map[string]interface{})
		terminatedStr := "terminated 🔵"

		running := "queued 🟣"
		if status["running"] != nil {
			running = terminatedStr
			if status["running"].(int) == 1 {
				running = "running 🟡"
			}
		}

		success := ""
		if running == terminatedStr {
			success = "without any output"
			if status["succeeded"] != nil && status["succeeded"].(int) == 1 {
				success = "succesfully 🟢"
			}
			if status["failed"] != nil && status["failed"].(int) == 1 {
				success = "with errors 🔴"
			}
		}

		message = fmt.Sprintf("Task %s %s", running, success)
	}

	if message == "logs" {
		logs := d.Get("logs").([]interface{})
		taskLogs := "No logs"
		if len(logs) > 0 {
			taskLogs = strings.Replace(logs[0].(string), "\n", fmt.Sprintf("\n[%s] ", levelText), -1)
		}

		message = fmt.Sprintf("Task logs:\x1b[%dm%s\x1b[0m", colors["foreground"], taskLogs)
	}

	tpl := "[%s] \x1b[%dm🚀TPI %s\x1b[0m %s'\n"
	return []byte(fmt.Sprintf(tpl, levelText, levelColor, d.Id(), message)), nil
}

func TpiLogger(d *schema.ResourceData) *logrus.Entry {
	logrus.SetFormatter(&tpiFormatter{})

	return logrus.WithFields(logrus.Fields{"d": d})
}
