package utils

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/acarl005/stripansi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sirupsen/logrus"
)

var baseTimestamp = time.Now()
var colors = make(map[string]int)

type basicFormatter struct{}

func (f *basicFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	levelText := strings.ToUpper(entry.Level.String())
	levelColor := colors[levelText]
	newPrefix := fmt.Sprintf("\x1b[%dmTPI [%s]\x1b[0m", levelColor, levelText)
	return []byte(hideUnwantedPrefix(levelText, newPrefix, entry.Message)), nil
}

func init() {
	colors["DEBUG"] = 34
	colors["INFO"] = 36
	colors["WARNING"] = 33
	colors["ERROR"] = 31
	colors["FATAL"] = 31
	colors["SUCCESS"] = 32
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
		return nil, errors.New("ResourceData is not available")
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

		message = fmt.Sprintf("\x1b[%dmStatus: queued \x1b[1m•\x1b[0m", colors["DEBUG"])

		if status["succeeded"] != nil && status["succeeded"].(int) >= d.Get("parallelism").(int) {
			message = fmt.Sprintf("\x1b[%dmStatus: completed succesfully \x1b[1m•\x1b[0m", colors["SUCCESS"])
		}
		if status["failed"] != nil && status["failed"].(int) > 0 {
			message = fmt.Sprintf("\x1b[%dmStatus: completed with errors \x1b[1m•\x1b[0m", colors["ERROR"])
		}
		if status["running"] != nil && status["running"].(int) >= d.Get("parallelism").(int) {
			message = fmt.Sprintf("\x1b[%dmStatus: running \x1b[1m•\x1b[0m", colors["WARNING"])
		}
	}

	if message == "logs" {
		message = ""
		logs := d.Get("logs").([]interface{})
		for index, log := range logs {
			prefix := fmt.Sprintf("\n\x1b[%dmLOG %d >> ", colors["foreground"], index)
			message += strings.Trim(strings.ReplaceAll("\n"+strings.Trim(log.(string), "\n"), "\n", prefix), "\n")
			if index+1 < len(logs) {
				message += "\n"
			}
		}
	}

	newPrefix := fmt.Sprintf("\x1b[%dmTPI [%s]\x1b[0m", levelColor, levelText)
	return []byte(hideUnwantedPrefix(levelText, newPrefix, message)), nil
}

func TpiLogger(d *schema.ResourceData) *logrus.Entry {
	logrus.SetFormatter(&tpiFormatter{})

	return logrus.WithFields(logrus.Fields{"d": d})
}

func hideUnwantedPrefix(levelText, newPrefix, message string) string {
	unwantedPrefixLength := len(fmt.Sprintf("yyyy-mm-ddThh:mm:ss.mmmZ [%s] provider.terraform-provider-iterative: [%[1]s]", levelText))

	var output string
	for _, line := range strings.Split(message, "\n") {
		formattedLine := fmt.Sprintf("[%s]\r%s %s", levelText, newPrefix, line)
		padding := strings.Repeat(" ", int(math.Max(float64(unwantedPrefixLength-len(stripansi.Strip(line))), 0)))
		output += formattedLine + padding + "\n"
	}

	return output
}
