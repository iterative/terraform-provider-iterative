package utils

import (
	"context"
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
