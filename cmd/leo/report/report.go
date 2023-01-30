package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"terraform-provider-iterative/task/common"
)

const (
	ClientName        = "leo"          // Name of the client sending the report
	StudioURLEnvKey   = "STUDIO_URL"   // Environment variable containing the URL of the Studio instance
	StudioTokenEnvKey = "STUDIO_TOKEN" // Environment variable containing the Studio token
)

var httpClient = &http.Client{
	Timeout: time.Second * 10,
}

type Options struct {
	StudioURL   string
	StudioToken string

	EventType   string
	RepoURL     string
	BaselineSHA string
	Name        string
	Client      string
	Step        string
	Params      []string
}

func New(cloud *common.Cloud) *cobra.Command {
	o := Options{}

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Send a studio live-metrics report.",
		Long: `
The report data is specified using command flags.

Repeated use of the --param flag adds free form data to the report's "params" field:
$ leo report -p key=value
{
...
 "params": {"key": "value"},
...
}

If the 'key' part contains periods, those are interpreted as traversing to object within
the 'params' field:
$ leo report -p key.subkey=value
{
...
  "params": {
    "key": {"subkey": "value"}
  }
...
}
`[1:],
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd, args, cloud)
		},
	}
	cmd.Flags().StringVar(&o.StudioURL, "url", "", "URL of the Studio instance")
	cmd.Flags().StringVar(&o.StudioToken, "token", "", "token for the Studio instance")

	cmd.Flags().StringVar(&o.EventType, "type", "", "event type")
	cmd.Flags().StringVar(&o.RepoURL, "repo", "", "URL of the repository the report is sent for")
	cmd.Flags().StringVar(&o.BaselineSHA, "sha", "", "Baseline sha for the repository")
	cmd.Flags().StringVar(&o.Name, "name", "", "name of the experiment being run ")
	cmd.Flags().StringVar(&o.Step, "step", "", "step of the experiment being run")
	cmd.Flags().StringArrayVarP(&o.Params, "param", "p", nil, "additional data, as key=value pairs")
	return cmd
}

// Run runs the subcommand.
func (o *Options) Run(cmd *cobra.Command, args []string, cloud *common.Cloud) error {
	studioURL := o.StudioURL
	if studioURL == "" {
		studioURL = os.Getenv(StudioURLEnvKey)
	}
	if studioURL == "" {
		return fmt.Errorf("URL of the studio instance not specified")
	}
	studioToken := o.StudioToken
	if studioToken == "" {
		studioToken = os.Getenv(StudioTokenEnvKey)
	}
	if studioToken == "" {
		return fmt.Errorf("Token for the studio instance not specified")
	}

	report, err := o.createReport()
	if err != nil {
		return err
	}

	body := &bytes.Buffer{}
	err = json.NewEncoder(body).Encode(report)
	if err != nil {
		return fmt.Errorf("failed to marshal to json: %w", err)
	}
	request, err := http.NewRequest(http.MethodPost, studioURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Add("Authorization", "token "+studioToken)
	request.Header.Add("Content-Type", "application/json")

	resp, err := httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send report: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := ioutil.ReadAll(resp.Body)
		if len(respBody) > 0 {
			return fmt.Errorf("unexpected Studio response (%d: %s): %q",
				resp.StatusCode,
				resp.Status,
				string(respBody))
		} else {
			return fmt.Errorf("unexpected Studio response %d: %s",
				resp.StatusCode,
				resp.Status)
		}
	}
	return nil
}

// createReport creates the base report structure
func (o *Options) createReport() (*StudioReport, error) {
	report := &StudioReport{
		EventType:   o.EventType,
		RepoURL:     o.RepoURL,
		BaselineSHA: o.BaselineSHA,
		Name:        o.Name,
		Step:        o.Step,
		Client:      ClientName,
	}
	params, err := paramsArrayToMap(o.Params)
	if err != nil {
		return nil, err
	}
	report.Params = params
	return report, nil
}

// StudioReport is marshalled to json and sent to the configured Iterative Studio
// instance.
type StudioReport struct {
	EventType   string      `json:"type"`
	RepoURL     string      `json:"repo_url,omitempty"`
	BaselineSHA string      `json:"baseline_sha,omitempty"`
	Name        string      `json:"name,omitempty"`
	Client      string      `json:"client,omitempty"`
	Step        string      `json:"step,omitempty"`
	Params      interface{} `json:"params"` // Free-form field.
}

// paramsArrayToMap converts and array of key=value pairs into a map containing
// the values. Keys with '.' separators are interpreted as pointing to maps within
// the parent map.
func paramsArrayToMap(params []string) (map[string]interface{}, error) {
	if len(params) == 0 {
		return nil, nil
	}
	result := map[string]interface{}{}
	for _, p := range params {
		keyValue := strings.SplitN(p, "=", 2)
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("invalid parameter %q, expecting 'key=value'", p)
		}
		key, value := keyValue[0], keyValue[1]
		keyParts := strings.Split(key, ".")
		target := result
		path := []string{}
		for _, keyPart := range keyParts[:len(keyParts)-1] {
			path = append(path, keyPart)
			entry, ok := target[keyPart]
			// The key does not exist in the map.
			if !ok {
				target[keyPart] = map[string]interface{}{}
				target = target[keyPart].(map[string]interface{})
				continue
			}
			if targetMap, ok := entry.(map[string]interface{}); ok {
				// The key exists in the map and points to a map[string]interface{}/
				target = targetMap
			} else {
				// Type mismatch, can happen when two conflicting parameter entries
				// are provided: 'key=value' and 'key.subkey=value'.
				return nil, fmt.Errorf("conflicting parameters %q and %q",
					strings.Join(path, "."), strings.Join(keyParts, "."))
			}
		}
		target[keyParts[len(keyParts)-1]] = value
	}
	return result, nil
}
