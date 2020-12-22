package iterative

import (
	"context"
	"fmt"
	"strconv"

	"terraform-provider-iterative/iterative/aws"
	"terraform-provider-iterative/iterative/azure"
	"terraform-provider-iterative/iterative/utils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/teris-io/shortid"
)

func resourceMachine() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMachineCreate,
		DeleteContext: resourceMachineDelete,
		ReadContext:   resourceMachineRead,
		UpdateContext: resourceMachineUpdate,
		Schema:        *machineSchema(),
	}
}

func machineSchema() *map[string]*schema.Schema {
	return &map[string]*schema.Schema{
		"cloud": &schema.Schema{
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
			Default:  "",
		},
		"region": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
			Default:  "us-west",
		},
		"image": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
			Default:  "",
		},
		"instance_name": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
			Default:  "",
		},
		"instance_type": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
			Default:  "m",
		},
		"instance_hdd_size": &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
			ForceNew: true,
			Default:  35,
		},
		"instance_gpu": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
			Default:  "",
		},
		"instance_id": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"instance_ip": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"instance_launch_time": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"ssh_public": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
			Default:  "",
		},
		"ssh_private": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"ssh_name": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
			Default:  "ubuntu",
		},
		"custom_data": &schema.Schema{
			Type:     schema.TypeString,
			ForceNew: true,
			Computed: true,
		},
		"aws_security_group": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
			Default:  "",
		},
	}
}

func resourceMachineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	maprefix := utils.MachinePrefix(d)
	hasMachine := len(maprefix) > 0

	keyPublic := d.Get(maprefix + "ssh_public").(string)
	if len(keyPublic) == 0 {
		public, private, _ := utils.SSHKeyPair()

		d.Set(maprefix+"ssh_public", public)
		d.Set(maprefix+"ssh_private", private)
	}

	sid, _ := shortid.New(1, shortid.DefaultABC, 2342)
	id, _ := sid.Generate()
	name := "iterative-" + id
	if hasMachine {

		d.Set("name", name)
		d.Set(maprefix+"instance_name", name)

	} else {
		instanceName := d.Get("instance_name").(string)
		if len(instanceName) == 0 {
			d.Set("instance_name", name)
		}
	}

	diags = append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("mapfrefix: %v", maprefix),
	})

	diags = append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  strconv.FormatBool(hasMachine),
	})

	diags = append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  d.Get(maprefix + "custom_data").(string),
	})

	return diags

	cloud := d.Get(maprefix + "cloud").(string)
	if cloud == "aws" {
		err := aws.ResourceMachineCreate(ctx, d, m)
		if err != nil {
			diags = append(resourceMachineDelete(ctx, d, m), diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Failed creating the machine: %v", err),
			})
		}
	} else if cloud == "azure" {
		err := azure.ResourceMachineCreate(ctx, d, m)
		if err != nil {
			diags = append(resourceMachineDelete(ctx, d, m), diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Failed creating the machine: %v", err),
			})

		}
	} else {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unknown cloud: %s", cloud),
		})
	}

	return diags
}

func resourceMachineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	maprefix := utils.MachinePrefix(d)

	cloud := d.Get(maprefix + "cloud").(string)
	if cloud == "aws" {
		err := aws.ResourceMachineDelete(ctx, d, m)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Failed disposing the machine: %v", err),
			})
		}
	} else if cloud == "azure" {
		err := azure.ResourceMachineDelete(ctx, d, m)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Failed disposing the machine: %v", err),
			})
		}
	}

	return diags
}

func resourceMachineRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func resourceMachineUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}
