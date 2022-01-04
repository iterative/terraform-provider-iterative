package utils

import (
	"os"

	"github.com/aohorodnyk/uid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

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

func LoadGCPCredentials() string {
	credentialsData := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_DATA")
	if len(credentialsData) == 0 {
		credentialsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		if len(credentialsPath) > 0 {
			jsonData, _ := os.ReadFile(credentialsPath)
			credentialsData = string(jsonData)
		}
	}
	return credentialsData
}
