package iterative

import (
	"context"
	"fmt"

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
		Schema: map[string]*schema.Schema{
			"cloud": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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

			"key_public": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "",
			},
			"key_private": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"key_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"aws_security_group": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "",
			},
		},
	}
}

func resourceMachineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	keyPublic := d.Get("key_public").(string)

	if len(keyPublic) == 0 {
		public, private, _ := utils.SSHKeyPair()

		d.Set("key_public", public)
		d.Set("key_private", private)
	}

	instanceName := d.Get("instance_name").(string)
	if len(instanceName) == 0 {
		sid, _ := shortid.New(1, shortid.DefaultABC, 2342)
		id, _ := sid.Generate()
		instanceName = "iterative-" + id

		d.Set("instance_name", instanceName)
	}

	cloud := d.Get("cloud").(string)

	if cloud == "aws" {
		return aws.ResourceMachineCreate(ctx, d, m)
	}

	if cloud == "azure" {
		return azure.ResourceMachineCreate(ctx, d, m)
	}

	diags = append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("Unknown cloud: %s", cloud),
	})
	return diags
}

func resourceMachineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	cloud := d.Get("cloud").(string)
	if cloud == "aws" {
		return aws.ResourceMachineDelete(ctx, d, m)
	}

	if cloud == "azure" {
		return azure.ResourceMachineDelete(ctx, d, m)
	}

	return nil
}

func resourceMachineRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func resourceMachineUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}
