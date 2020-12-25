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

func SetId(d *schema.ResourceData) {
	sid, _ := shortid.New(1, shortid.DefaultABC, 2342)
	id, _ := sid.Generate()
	name := "iterative-" + id
	d.SetId(name)

	if len(d.Get("name").(string)) == 0 {
		d.Set("name", name)
	}
}
