package utils

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/teris-io/shortid"
)

func MachinePrefix(d *schema.ResourceData) string {
	prefix := ""
	if _, hasMachine := d.GetOk("machine"); hasMachine {
		prefix = "machine.0."
	}

	return prefix
}

func SetName(d *schema.ResourceData) {
	name := d.Get("name").(string)
	if len(name) == 0 {
		sid, _ := shortid.New(1, shortid.DefaultABC, 2342)
		id, _ := sid.Generate()
		d.Set("name", "iterative-"+id)
	}
}
