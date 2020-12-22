package iterative

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strconv"
	"terraform-provider-iterative/iterative/utils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRunner() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRunnerCreate,
		DeleteContext: resourceRunnerDelete,
		ReadContext:   resourceRunnerRead,
		UpdateContext: resourceRunnerUpdate,
		Schema: map[string]*schema.Schema{
			"repo": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"token": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"driver": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"labels": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "cml",
			},
			"idle_timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  300,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "",
			},
			"machine": &schema.Schema{
				Optional: true,
				Type:     schema.TypeList,
				Elem: &schema.Resource{
					Schema: *machineSchema(),
				},
			},
		},
	}
}

func resourceRunnerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	maprefix := utils.MachinePrefix(d)

	customData, err := provisionerCode(d)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error generating provisioner code: %s", err),
		})
		return diags
	}
	d.Set(maprefix+"custom_data", customData)

	cloud := d.Get(maprefix + "cloud").(string)
	if len(cloud) == 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Local runner not yet implemented"),
		})
	} else {
		diags = resourceMachineCreate(ctx, d, m)
	}

	return diags
}

func resourceRunnerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceMachineDelete(ctx, d, m)
}

func resourceRunnerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func resourceRunnerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func provisionerCode(d *schema.ResourceData) (string, error) {
	var code string

	data := make(map[string]string)
	data["token"] = d.Get("token").(string)
	data["repo"] = d.Get("repo").(string)
	data["driver"] = d.Get("driver").(string)
	data["labels"] = d.Get("labels").(string)
	data["idle_timeout"] = strconv.Itoa(d.Get("idle_timeout").(int))
	data["name"] = d.Get("name").(string)

	tmpl, err := template.New("deploy").Parse(`#!/bin/bash
echo "APT::Get::Assume-Yes \"true\";" | sudo tee -a /etc/apt/apt.conf.d/90assumeyes
curl -sL https://deb.nodesource.com/setup_12.x | sudo bash
curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -
sudo apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"
sudo apt update && sudo apt-get install -y terraform nodejs
sudo npm install -g git+https://github.com/iterative/cml.git#cml-runner
nohup cml-runner{{if .name}} --name {{.name}}{{end}}{{if .labels}} --labels {{.labels}}{{end}}{{if .idle_timeout}} --idle-timeout {{.idle_timeout}}{{end}}{{if .driver}} --driver {{.driver}}{{end}}{{if .repo}} --repo {{.repo}}{{end}}{{if .token}} --token {{.token}}{{end}} < /dev/null > std.out 2> std.err &
sleep 10
`)
	var customDataBuffer bytes.Buffer
	err = tmpl.Execute(&customDataBuffer, data)

	if err == nil {
		code = customDataBuffer.String()
	}

	return code, nil
}
