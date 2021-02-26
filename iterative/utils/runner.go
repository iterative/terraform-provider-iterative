package utils

import (
	"bufio"
	"encoding/json"
	"regexp"
	"strings"
)

type LogEvent struct {
	Level      string `json:"level"`
	Time       string `json:"time"`
	Repository string `json:"repo"`
	Job        string `json:"job"`
	Status     string `json:"status"`
	Success    bool   `json:"success"`
}

func ParseLogEvent(logEvent string) (LogEvent, error) {
	var result LogEvent
	err := json.Unmarshal([]byte(logEvent), &result)
	return result, err
}

// IsReady checks whether a runner is ready or not by
// parsing the JSONL records from the logs it produces.
func IsReady(logs string) bool {
	scanner := bufio.NewScanner(strings.NewReader(logs))
	for scanner.Scan() {
		line := scanner.Text()
		// Extract the JSON between curly braces from the log line.
		record := regexp.MustCompile(`\{.+\}`).Find([]byte(line))
		// Try to parse the retrieved JSON string into a LogEvent structure.
		if event, err := ParseLogEvent(string(record)); err == nil {
			if event.Status == "ready" {
				return true
			}
		}
	}
	return false
}
