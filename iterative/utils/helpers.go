package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aohorodnyk/uid"
	"github.com/blang/semver/v4"
	"github.com/google/go-github/v42/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func GetCML(version string) string {
	// default latest if unset
	if version == "" {
		client := github.NewClient(nil)
		release, _, err := client.Repositories.GetLatestRelease(context.Background(), "iterative", "cml")
		if err != nil {
			// GitHub API failed
			return getNPMCML("@dvcorg/cml")
		}
		for _, asset := range release.Assets {
			if *asset.Name == "cml-linux" {
				return getGHCML(*asset.BrowserDownloadURL)
			}
		}
	}
	// handle semver
	ver, err := semver.Make(strings.TrimPrefix(version, "v"))
	if err == nil {
		return getSemverCML(ver)
	}
	// user must know best, npm install <string>
	if version != "" {
		return getNPMCML(version)
	}
	// original fallback, some error has forced this
	return getNPMCML("@dvcorg/cml")
}

func getGHCML(v string) string {
	return fmt.Sprintf(`sudo mkdir -p /opt/cml/
sudo curl --location --url %s --output /opt/cml/cml-linux
sudo chmod +x /opt/cml/cml-linux
sudo ln -s /opt/cml/cml-linux /usr/bin/cml
sudo ln /opt/cml/cml-linux /usr/bin/cml-internal`, v) // hard link to fix cml#920
}

func getNPMCML(v string) string {
	npmCML := "sudo npm config set user 0 && sudo npm install --global %s"
	return fmt.Sprintf(npmCML, v)
}

func getSemverCML(sv semver.Version) string {
	directDownloadVersion, _ := semver.ParseRange(">=0.10.0")
	if directDownloadVersion(sv) {
		client := github.NewClient(nil)
		release, _, err := client.Repositories.GetReleaseByTag(context.Background(), "iterative", "cml", "v"+sv.String())
		if err == nil {
			for _, asset := range release.Assets {
				if *asset.Name == "cml-linux" {
					return getGHCML(*asset.BrowserDownloadURL)
				}
			}
		}
	}
	// npm install
	return getNPMCML("@dvcorg/cml@v" + sv.String())
}

func MachinePrefix(d *schema.ResourceData) string {
	prefix := ""
	if _, hasMachine := d.GetOk("machine"); hasMachine {
		prefix = "machine.0."
	}

	return prefix
}

func SetId(d *schema.ResourceData) {
	if len(d.Id()) == 0 {
		d.SetId("iterative-" + uid.NewProvider36Size(8).MustGenerate().String())

		if len(d.Get("name").(string)) == 0 {
			d.Set("name", d.Id())
		}
	}
}

func MultiEnvLoadFirst(envs []string) string {
	for _, val := range envs {
		if env_value := os.Getenv(val); env_value != "" {
			return env_value
		}
	}
	return ""
}

func GCPCoerceOIDCCredentials(credentialsJSON []byte) (string, error) {
	var credentials map[string]interface{}
	if err := json.Unmarshal(credentialsJSON, &credentials); err != nil {
		return "", err
	}

	if url, ok := credentials["service_account_impersonation_url"].(string); ok {
		re := regexp.MustCompile("^https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/.+?@(?P<project>.+).iam.gserviceaccount.com:generateAccessToken$")
		if match := re.FindStringSubmatch(url); match != nil {
			return match[1], nil
		}
		return "", errors.New("failed to get project identifier from service_account_impersonation_url")
	}

	return "", errors.New("unable to load service_account_impersonation_url")
}

// Better way than copying?
// https://github.com/hashicorp/terraform-provider-google/blob/8a362008bd4d36b6a882eb53455f87305e6dff52/google/service_scope.go#L5-L48
func canonicalizeServiceScope(scope string) string {
	// This is a convenience map of short names used by the gcloud tool
	// to the GCE auth endpoints they alias to.
	scopeMap := map[string]string{
		"bigquery":              "https://www.googleapis.com/auth/bigquery",
		"cloud-platform":        "https://www.googleapis.com/auth/cloud-platform",
		"cloud-source-repos":    "https://www.googleapis.com/auth/source.full_control",
		"cloud-source-repos-ro": "https://www.googleapis.com/auth/source.read_only",
		"compute-ro":            "https://www.googleapis.com/auth/compute.readonly",
		"compute-rw":            "https://www.googleapis.com/auth/compute",
		"datastore":             "https://www.googleapis.com/auth/datastore",
		"logging-write":         "https://www.googleapis.com/auth/logging.write",
		"monitoring":            "https://www.googleapis.com/auth/monitoring",
		"monitoring-read":       "https://www.googleapis.com/auth/monitoring.read",
		"monitoring-write":      "https://www.googleapis.com/auth/monitoring.write",
		"pubsub":                "https://www.googleapis.com/auth/pubsub",
		"service-control":       "https://www.googleapis.com/auth/servicecontrol",
		"service-management":    "https://www.googleapis.com/auth/service.management.readonly",
		"sql":                   "https://www.googleapis.com/auth/sqlservice",
		"sql-admin":             "https://www.googleapis.com/auth/sqlservice.admin",
		"storage-full":          "https://www.googleapis.com/auth/devstorage.full_control",
		"storage-ro":            "https://www.googleapis.com/auth/devstorage.read_only",
		"storage-rw":            "https://www.googleapis.com/auth/devstorage.read_write",
		"taskqueue":             "https://www.googleapis.com/auth/taskqueue",
		"trace":                 "https://www.googleapis.com/auth/trace.append",
		"useraccounts-ro":       "https://www.googleapis.com/auth/cloud.useraccounts.readonly",
		"useraccounts-rw":       "https://www.googleapis.com/auth/cloud.useraccounts",
		"userinfo-email":        "https://www.googleapis.com/auth/userinfo.email",
	}
	if matchedURL, ok := scopeMap[scope]; ok {
		return matchedURL
	}
	return scope
}
func CanonicalizeServiceScopes(scopes []string) []string {
	cs := make([]string, len(scopes))
	for i, scope := range scopes {
		cs[i] = canonicalizeServiceScope(scope)
	}
	return cs
}
