package iterative

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRunner() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRunnerCreate,
		DeleteContext: resourceRunnerDelete,
		ReadContext:   resourceMachineRead,
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
				Computed: true,
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
			"instance_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"instance_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"instance_launch_time": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"ssh_public": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
			"ssh_private": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
			"ssh_name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "ubuntu",
			},
			"custom_data": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"aws_security_group": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
		},
	}
}

func resourceRunnerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	customData, err := provisionerCode(d)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error generating provisioner code: %s", err),
		})
		return diags
	}
	d.Set("custom_data", customData)

	diags = append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  d.Get("custom_data").(string),
	})

	return diags

	cloud := d.Get("cloud").(string)
	if len(cloud) == 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Local runner not yet implemented"),
		})
	} else {
		diags = resourceMachineCreate(ctx, d, m)
		//d.SetId("local")
	}

	return diags
}

func resourceRunnerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceMachineDelete(ctx, d, m)
}

func provisionerCode(d *schema.ResourceData) (string, error) {
	var code string

	tfResource := ResourceType{
		Mode:     "managed",
		Type:     "iterative_runner",
		Name:     "runner",
		Provider: "provider[\"registry.terraform.io/iterative/iterative\"]",
		Instances: InstancesType{
			InstanceType{
				SchemaVersion: 0,
				Attributes: AttributesType{
					ID:                 d.Get("name").(string),
					Cloud:              d.Get("cloud").(string),
					Name:               "",
					Labels:             "",
					IdleTimeout:        d.Get("idle_timeout").(int),
					Repo:               "",
					Token:              "",
					Driver:             "",
					AwsSecurityGroup:   "",
					CustomData:         "",
					Image:              "",
					InstanceGpu:        "",
					InstanceHddSize:    d.Get("instance_hdd_size").(int),
					InstanceID:         "",
					InstanceIP:         "",
					InstanceLaunchTime: "",
					InstanceType:       "",
					Region:             "",
					SSHName:            "",
					SSHPrivate:         "",
					SSHPublic:          "",
				},
			},
		},
	}
	jsonResource, err := json.Marshal(tfResource)
	if err != nil {
		return code, err
	}

	data := make(map[string]string)
	data["token"] = d.Get("token").(string)
	data["repo"] = d.Get("repo").(string)
	data["driver"] = d.Get("driver").(string)
	data["labels"] = d.Get("labels").(string)
	data["idle_timeout"] = strconv.Itoa(d.Get("idle_timeout").(int))
	data["name"] = d.Get("name").(string)
	data["tf_resource"] = base64.StdEncoding.EncodeToString(jsonResource)
	data["AWS_SECRET_ACCESS_KEY"] = os.Getenv("AWS_SECRET_ACCESS_KEY")
	data["AWS_ACCESS_KEY_ID"] = os.Getenv("AWS_ACCESS_KEY_ID")
	data["AZURE_CLIENT_ID"] = os.Getenv("AZURE_CLIENT_ID")
	data["AZURE_CLIENT_SECRET"] = os.Getenv("AZURE_CLIENT_SECRET")
	data["AZURE_SUBSCRIPTION_ID"] = os.Getenv("AZURE_SUBSCRIPTION_ID")
	data["AZURE_TENANT_ID"] = os.Getenv("AZURE_TENANT_ID")

	tmpl, err := template.New("deploy").Parse(`#!/bin/bash
export DEBIAN_FRONTEND=noninteractive
echo "APT::Get::Assume-Yes \"true\";" | sudo tee -a /etc/apt/apt.conf.d/90assumeyes
curl -sL https://deb.nodesource.com/setup_12.x | sudo bash
curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -
sudo apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"
sudo apt update && sudo apt-get install -y terraform nodejs
sudo npm install -g git+https://github.com/iterative/cml.git#cml-runner

export AWS_SECRET_ACCESS_KEY={{.AWS_SECRET_ACCESS_KEY}}
export AWS_ACCESS_KEY_ID={{.AWS_ACCESS_KEY_ID}}
export AZURE_CLIENT_ID={{.AZURE_CLIENT_ID}}
export AZURE_CLIENT_SECRET={{.AZURE_CLIENT_SECRET}}
export AZURE_SUBSCRIPTION_ID={{.AZURE_SUBSCRIPTION_ID}}
export AZURE_TENANT_ID={{.AZURE_TENANT_ID}}
nohup cml-runner{{if .name}} --name {{.name}}{{end}}{{if .labels}} --labels {{.labels}}{{end}}{{if .idle_timeout}} --idle-timeout {{.idle_timeout}}{{end}}{{if .driver}} --driver {{.driver}}{{end}}{{if .repo}} --repo {{.repo}}{{end}}{{if .token}} --token {{.token}}{{end}}{{if .tf_resource}} --tf_resource={{.tf_resource}}{{end}} < /dev/null > std.out 2> std.err &
sleep 10
`)
	var customDataBuffer bytes.Buffer
	err = tmpl.Execute(&customDataBuffer, data)

	if err == nil {
		code = customDataBuffer.String()
	}

	return code, nil
}

type AttributesType struct {
	Name               string      `json:"name"`
	Labels             string      `json:"labels"`
	IdleTimeout        int         `json:"idle_timeout"`
	Repo               string      `json:"repo"`
	Token              string      `json:"token"`
	Driver             string      `json:"driver"`
	AwsSecurityGroup   interface{} `json:"aws_security_group"`
	Cloud              string      `json:"cloud"`
	CustomData         string      `json:"custom_data"`
	ID                 string      `json:"id"`
	Image              interface{} `json:"image"`
	InstanceGpu        interface{} `json:"instance_gpu"`
	InstanceHddSize    int         `json:"instance_hdd_size"`
	InstanceID         string      `json:"instance_id"`
	InstanceIP         string      `json:"instance_ip"`
	InstanceLaunchTime string      `json:"instance_launch_time"`
	InstanceType       string      `json:"instance_type"`
	Region             string      `json:"region"`
	SSHName            string      `json:"ssh_name"`
	SSHPrivate         string      `json:"ssh_private"`
	SSHPublic          string      `json:"ssh_public"`
}
type InstanceType struct {
	Private       string         `json:"private"`
	SchemaVersion int            `json:"schema_version"`
	Attributes    AttributesType `json:"attributes"`
}
type InstancesType []InstanceType
type ResourceType struct {
	Mode      string        `json:"mode"`
	Type      string        `json:"type"`
	Name      string        `json:"name"`
	Provider  string        `json:"provider"`
	Instances InstancesType `json:"instances"`
}
