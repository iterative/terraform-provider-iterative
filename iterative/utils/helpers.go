package utils

import (
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

func StripAvailabilityZone(region string) string {
	lastChar := region[len(region)-1]
	// 0x61(a) to 0x7a(z)
	if lastChar >= 0x61 && lastChar <= 0x71 {
		return region[:len(region)-1]
	} else {
		return region
	}
}
