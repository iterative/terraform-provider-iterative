package iterative

import (
	"context"
	"encoding/base64"
	"fmt"

	"terraform-provider-iterative/iterative/aws"
	"terraform-provider-iterative/iterative/azure"
	"terraform-provider-iterative/iterative/utils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceMachine() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMachineCreate,
		DeleteContext: resourceMachineDelete,
		ReadContext:   resourceMachineRead,
		Schema:        *machineSchema(),
	}
}

func machineSchema() *map[string]*schema.Schema {
	return &map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
			Default:  "",
		},
		"cloud": &schema.Schema{
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
			Default:  "",
		},
		"region": &schema.Schema{
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
			Default:  "us-west",
		},
		"image": &schema.Schema{
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
			Default:  "",
		},
		"spot": &schema.Schema{
			Type:     schema.TypeBool,
			ForceNew: true,
			Optional: true,
			Default:  false,
		},
		"spotPrice": &schema.Schema{
			Type:     schema.TypeFloat,
			ForceNew: true,
			Optional: true,
			Default:  -1,
		},
		"instance_type": &schema.Schema{
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
			Default:  "m",
		},
		"instance_hdd_size": &schema.Schema{
			Type:     schema.TypeInt,
			ForceNew: true,
			Optional: true,
			Default:  35,
		},
		"instance_gpu": &schema.Schema{
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
			Default:  "",
		},
		"instance_ip": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
		"instance_launch_time": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
		"ssh_private": &schema.Schema{
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
			Default:  "",
		},
		"ssh_name": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
		"startup_script": &schema.Schema{
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
			Default:  "#!/bin/bash",
		},
		"aws_security_group": &schema.Schema{
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
			Default:  "",
		},
	}
}

func resourceMachineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	utils.SetId(d)

	if len(d.Get("ssh_private").(string)) == 0 {
		private, err := utils.PrivatePEM()
		if err != nil {
			diags = append(resourceMachineDelete(ctx, d, m), diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Failed creating the private PEM: %v", err),
			})

			return diags
		}

		d.Set("ssh_private", private)
	}

	public, err := utils.PublicFromPrivatePEM(d.Get("ssh_private").(string))
	if err != nil {
		diags = append(resourceMachineDelete(ctx, d, m), diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Failed creating the public key: %v", err),
		})

		return diags
	}
	d.Set("ssh_public", public)

	script64 := base64.StdEncoding.EncodeToString([]byte(d.Get("startup_script").(string)))
	d.Set("startup_script", script64)

	cloud := d.Get("cloud").(string)
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

	cloud := d.Get("cloud").(string)
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
