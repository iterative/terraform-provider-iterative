package machine

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/alessio/shellescape"

	"github.com/iterative/terraform-provider-iterative/task/common"
)

//go:embed machine-script.sh.tpl
var machineScript string

var machineScriptTemplate = template.Must(template.New("machine-script").Parse(machineScript))

func Script(script string, credentials map[string]string, variables common.Variables, timeout *time.Time) (string, error) {
	timeoutString := "infinity"
	if timeout != nil {
		timeoutString = fmt.Sprintf("%d", timeout.Unix())
	}

	environment := ""
	for name, value := range variables.Enrich() {
		escaped := strings.ReplaceAll(value, `"`, `\"`) // FIXME: \" edge cases.
		environment += fmt.Sprintf("%s=\"%s\"\n", name, escaped)
	}

	exportCredentials := ""
	for name, value := range credentials {
		exportCredentials += "export " + shellescape.Quote(name+"="+value) + "\n"
	}

	var output bytes.Buffer
	machineScriptParams := struct {
		TaskScript  string
		Environment string
		Credentials string
		Timeout     string
	}{
		TaskScript:  base64.StdEncoding.EncodeToString([]byte(script)),
		Environment: base64.StdEncoding.EncodeToString([]byte(environment)),
		Credentials: base64.StdEncoding.EncodeToString([]byte(exportCredentials)),
		Timeout:     timeoutString,
	}

	err := machineScriptTemplate.Execute(&output, machineScriptParams)
	if err != nil {
		return "", err
	}
	return output.String(), nil
}
