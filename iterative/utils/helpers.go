package utils

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func MachinePrefix(d *schema.ResourceData) string {
	prefix := ""
	if _, hasMachine := d.GetOk("machine"); hasMachine {
		prefix = "machine.0."
	}

	return prefix
}
