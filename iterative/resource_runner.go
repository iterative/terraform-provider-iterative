package iterative

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"text/template"
	"time"

	"gopkg.in/alessio/shellescape.v1"

	"terraform-provider-iterative/iterative/aws"
	"terraform-provider-iterative/iterative/azure"
	"terraform-provider-iterative/iterative/utils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRunner() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRunnerCreate,
		DeleteContext: resourceRunnerDelete,
		ReadContext:   resourceMachineRead,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"repo": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"token": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "",
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
			"single": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
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
			"spot": &schema.Schema{
				Type:     schema.TypeBool,
				ForceNew: true,
				Optional: true,
				Default:  false,
			},
			"spot_price": &schema.Schema{
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
			"ssh_public": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"ssh_private": &schema.Schema{
				Type:      schema.TypeString,
				ForceNew:  true,
				Optional:  true,
				Default:   "",
				Sensitive: true,
			},
			"startup_script": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
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

	utils.SetId(d)

	token := os.Getenv("CML_TOKEN")
	if len(d.Get("token").(string)) == 0 && len(token) != 0 {
		d.Set("token", token)
	}

	if len(d.Get("token").(string)) == 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Token not found nor in tf file nor in env CML_TOKEN"),
		})
		return diags
	}

	startupScript, err := provisionerCode(d)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error generating startup script: %s", err),
		})
		return diags
	}

	d.Set("startup_script", startupScript)
	if d.Get("instance_gpu") == "tesla" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("GPU model 'tesla' has been deprecated; please use 'v100' instead"),
		})
		d.Set("instance_gpu", "v100")
	}

	if len(d.Get("cloud").(string)) == 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Local runner not yet implemented"),
		})
	} else {
		diags = resourceMachineCreate(ctx, d, m)
	}

	if diags.HasError() {
		return diags
	}
	log.Printf("[DEBUG] Instance address: %#v", d.Get("instance_ip"))

	var logError error
	var logEvents string
	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		logEvents, logError = resourceMachineLogs(ctx, d, m)
		log.Printf("[DEBUG] Collected log events: %#v", logEvents)
		log.Printf("[DEBUG] Connection errors: %#v", logError)

		if logError != nil {
			return resource.RetryableError(fmt.Errorf("Waiting for the machine to accept connections... %s", logError))
		} else if utils.HasStatus(logEvents, "terminated") {
			return resource.NonRetryableError(fmt.Errorf("Failed to launch the runner!"))
		} else if utils.HasStatus(logEvents, "ready") {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("Waiting for the runner to be ready..."))
	})

	if logError != nil {
		logEvents += "\n" + logError.Error()
	}

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error checking the runner status"),
			Detail:   logEvents,
		})
	}

	return diags
}

func resourceRunnerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceMachineDelete(ctx, d, m)
}

func renderScript(data map[string]interface{}) (string, error) {
	var script string

	tmpl, err := template.New("deploy").Funcs(template.FuncMap{"escape": shellescape.Quote}).Parse(
		`#!/bin/sh
{{if not (or .ami .container)}}
export DEBIAN_FRONTEND=noninteractive
echo "APT::Get::Assume-Yes \"true\";" | sudo tee -a /etc/apt/apt.conf.d/90assumeyes

sudo apt remove unattended-upgrades
systemctl disable apt-daily-upgrade.service

sudo add-apt-repository universe -y
sudo add-apt-repository ppa:git-core/ppa -y
sudo apt update && sudo apt install -y software-properties-common git build-essential

sudo curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh
sudo usermod -aG docker ubuntu
sudo setfacl --modify user:ubuntu:rw /var/run/docker.sock

curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -
sudo apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"
sudo apt update && sudo apt-get install -y terraform

curl -sL https://deb.nodesource.com/setup_12.x | sudo bash
sudo apt update && sudo apt-get install -y nodejs

sudo apt install ubuntu-drivers-common
sudo ubuntu-drivers autoinstall
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/ubuntu18.04/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list
sudo apt update && sudo apt install -y nvidia-docker2
sudo systemctl restart docker
sudo nvidia-smi
sudo docker run --rm --gpus all nvidia/cuda:11.0-base nvidia-smi
{{end}}

{{if not .container}}
sudo npm config set user 0 && sudo npm install --global @dvcorg/cml
{{end}}

{{if .runner_startup_script}}
{{.runner_startup_script}}
{{end}}

{{if not .container}}
sudo bash -c 'cat << EOF > /usr/bin/cml.sh
#!/bin/sh
{{end}}

{{if .cloud}}
{{if eq .cloud "aws"}}
export AWS_SECRET_ACCESS_KEY={{escape .AWS_SECRET_ACCESS_KEY}}
export AWS_ACCESS_KEY_ID={{escape .AWS_ACCESS_KEY_ID}}
export AWS_SESSION_TOKEN={{escape .AWS_SESSION_TOKEN}}
{{end}}
{{if eq .cloud "azure"}}
export AZURE_CLIENT_ID={{escape .AZURE_CLIENT_ID}}
export AZURE_CLIENT_SECRET={{escape .AZURE_CLIENT_SECRET}}
export AZURE_SUBSCRIPTION_ID={{escape .AZURE_SUBSCRIPTION_ID}}
export AZURE_TENANT_ID={{escape .AZURE_TENANT_ID}}
{{end}}
{{if eq .cloud "kubernetes"}}
export KUBERNETES_CONFIGURATION={{escape .KUBERNETES_CONFIGURATION}}
{{end}}
{{end}}

HOME="$(mktemp -d)" exec cml-runner \
  {{if .name}} --name {{escape .name}}{{end}} \
  {{if .labels}} --labels {{escape .labels}}{{end}} \
  {{if .idle_timeout}} --idle-timeout {{escape .idle_timeout}}{{end}} \
  {{if .driver}} --driver {{escape .driver}}{{end}} \
  {{if .repo}} --repo {{escape .repo}}{{end}} \
  {{if .token}} --token {{escape .token}}{{end}} \
  {{if .single}} --single{{end}} \
  {{if .tf_resource}} --tf-resource {{escape .tf_resource}}{{end}}

{{- if not .container}}
EOF'
sudo chmod +x /usr/bin/cml.sh

sudo bash -c 'cat << EOF > /etc/systemd/system/cml.service
[Unit]
  Description=cml service

[Service]
  Type=oneshot
  RemainAfterExit=yes
  ExecStart=/usr/bin/cml.sh

[Install]
  WantedBy=multi-user.target
EOF'

sudo systemctl daemon-reload
sudo systemctl enable cml.service --now
{{- end}}
`)
	var customDataBuffer bytes.Buffer
	err = tmpl.Execute(&customDataBuffer, data)

	if err == nil {
		script = customDataBuffer.String()
	}

	return script, err
}

func provisionerCode(d *schema.ResourceData) (string, error) {
	var code string

	tfResource := ResourceType{
		Mode:     "managed",
		Type:     "iterative_cml_runner",
		Name:     "runner",
		Provider: "provider[\"registry.terraform.io/iterative/iterative\"]",
		Instances: InstancesType{
			InstanceType{
				SchemaVersion: 0,
				Attributes: AttributesType{
					ID:                 d.Id(),
					Cloud:              d.Get("cloud").(string),
					Region:             d.Get("region").(string),
					Name:               d.Get("name").(string),
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
					InstanceIP:         "",
					InstanceLaunchTime: "",
					InstanceType:       "",
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

	data := make(map[string]interface{})
	data["token"] = d.Get("token").(string)
	data["repo"] = d.Get("repo").(string)
	data["driver"] = d.Get("driver").(string)
	data["labels"] = d.Get("labels").(string)
	data["idle_timeout"] = strconv.Itoa(d.Get("idle_timeout").(int))
	data["name"] = d.Get("name").(string)
	data["cloud"] = d.Get("cloud").(string)
	data["tf_resource"] = base64.StdEncoding.EncodeToString(jsonResource)
	data["instance_gpu"] = d.Get("instance_gpu").(string)
	data["single"] = d.Get("single").(bool)
	data["AWS_SECRET_ACCESS_KEY"] = os.Getenv("AWS_SECRET_ACCESS_KEY")
	data["AWS_ACCESS_KEY_ID"] = os.Getenv("AWS_ACCESS_KEY_ID")
	data["AWS_SESSION_TOKEN"] = os.Getenv("AWS_SESSION_TOKEN")
	data["AZURE_CLIENT_ID"] = os.Getenv("AZURE_CLIENT_ID")
	data["AZURE_CLIENT_SECRET"] = os.Getenv("AZURE_CLIENT_SECRET")
	data["AZURE_SUBSCRIPTION_ID"] = os.Getenv("AZURE_SUBSCRIPTION_ID")
	data["AZURE_TENANT_ID"] = os.Getenv("AZURE_TENANT_ID")
	data["KUBERNETES_CONFIGURATION"] = os.Getenv("KUBERNETES_CONFIGURATION")
	data["ami"] = isAMIAvailable(d.Get("cloud").(string), d.Get("region").(string))
	data["container"] = isContainerAvailable(d.Get("cloud").(string))
	script, err := base64.StdEncoding.DecodeString(d.Get("startup_script").(string))
	if err != nil {
		return "", err
	}

	data["runner_startup_script"] = string(script)

	return renderScript(data)
}

func isAMIAvailable(cloud string, region string) bool {
	regions := aws.ImageRegions
	if cloud == "azure" {
		regions = azure.ImageRegions
	}

	for _, item := range regions {
		if item == region {
			return true
		}
	}

	return false
}

func isContainerAvailable(cloud string) bool {
	switch cloud {
	case "kubernetes":
		return true
	default:
		return false
	}
}

type AttributesType struct {
	Name               string      `json:"name"`
	Labels             string      `json:"labels"`
	IdleTimeout        int         `json:"idle_timeout"`
	Repo               string      `json:"repo"`
	Token              string      `json:"token"`
	Driver             string      `json:"driver"`
	Cloud              string      `json:"cloud"`
	CustomData         string      `json:"custom_data"`
	ID                 string      `json:"id"`
	Image              interface{} `json:"image"`
	InstanceGpu        interface{} `json:"instance_gpu"`
	InstanceHddSize    int         `json:"instance_hdd_size"`
	InstanceIP         string      `json:"instance_ip"`
	InstanceLaunchTime string      `json:"instance_launch_time"`
	InstanceType       string      `json:"instance_type"`
	Region             string      `json:"region"`
	SSHName            string      `json:"ssh_name"`
	SSHPrivate         string      `json:"ssh_private"`
	SSHPublic          string      `json:"ssh_public"`
	AwsSecurityGroup   interface{} `json:"aws_security_group"`
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
