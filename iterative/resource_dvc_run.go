package iterative

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"terraform-provider-iterative/iterative/utils"
	"text/template"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"gopkg.in/alessio/shellescape.v1"
)

func resourceDVCRun() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDVCRunCreate,
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
			"dvc_ver": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Default:  "latest",
			},
			"labels": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "dvc",
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
				Type:      schema.TypeString,
				ForceNew:  true,
				Optional:  true,
				Default:   "",
				Sensitive: true,
			},
			"aws_security_group": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				ForceNew: true,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceDVCRunCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	utils.SetId(d)

	startupScript, err := dvcRunProvisioner(d)
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
		switch cloud := d.Get("cloud").(string); cloud {
		case "kubernetes":
			logEvents, logError = resourceMachineLogs(ctx, d, m)
		default:
			logEvents, logError = utils.RunCommand("journalctl --unit dvc-run --no-pager",
				2*time.Second,
				net.JoinHostPort(d.Get("instance_ip").(string), "22"),
				"ubuntu",
				d.Get("ssh_private").(string))
		}

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

func dvcRunProvisioner(d *schema.ResourceData) (string, error) {
	data := make(map[string]interface{})
	data["repo"] = d.Get("repo").(string)
	data["dvc_ver"] = d.Get("dvc_ver").(string)
	data["labels"] = d.Get("labels").(string)
	data["idle_timeout"] = strconv.Itoa(d.Get("idle_timeout").(int))
	data["name"] = d.Get("name").(string)
	data["cloud"] = d.Get("cloud").(string)
	data["startup_script"] = d.Get("startup_script").(string)
	data["instance_gpu"] = d.Get("instance_gpu").(string)
	data["single"] = d.Get("single").(bool)
	data["AWS_SECRET_ACCESS_KEY"] = os.Getenv("AWS_SECRET_ACCESS_KEY")
	data["AWS_ACCESS_KEY_ID"] = os.Getenv("AWS_ACCESS_KEY_ID")
	data["AWS_SESSION_TOKEN"] = os.Getenv("AWS_SESSION_TOKEN")
	data["AZURE_CLIENT_ID"] = os.Getenv("AZURE_CLIENT_ID")
	data["AZURE_CLIENT_SECRET"] = os.Getenv("AZURE_CLIENT_SECRET")
	data["AZURE_SUBSCRIPTION_ID"] = os.Getenv("AZURE_SUBSCRIPTION_ID")
	data["AZURE_TENANT_ID"] = os.Getenv("AZURE_TENANT_ID")
	data["GOOGLE_APPLICATION_CREDENTIALS_DATA"] = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_DATA")
	data["KUBERNETES_CONFIGURATION"] = os.Getenv("KUBERNETES_CONFIGURATION")
	data["container"] = isContainerAvailable(d.Get("cloud").(string))

	return renderDVCScript(data)
}

func renderDVCScript(data map[string]interface{}) (string, error) {
	var script string
	startup_script, ok := data["startup_script"].(string)
	if ok {
		runnerStartupScript, err := base64.StdEncoding.DecodeString(startup_script)
		if err != nil {
			return script, err
		}

		data["runner_startup_script"] = string(runnerStartupScript)
	}

	tmpl, err := template.New("deploy").Funcs(template.FuncMap{"escape": shellescape.Quote}).Parse(
		`#!/bin/sh
{{- if .runner_startup_script}}
{{.runner_startup_script}}
{{- end}}

{{- if not .container}}
sudo tee /usr/bin/dvc_run.sh << 'EOF'
#!/bin/sh
sudo apt-get install -y python3-pip;
{{- end}}

{{- if .cloud}}
{{- if eq .cloud "aws"}}
export AWS_SECRET_ACCESS_KEY={{escape .AWS_SECRET_ACCESS_KEY}}
export AWS_ACCESS_KEY_ID={{escape .AWS_ACCESS_KEY_ID}}
export AWS_SESSION_TOKEN={{escape .AWS_SESSION_TOKEN}}
{{- end}}
{{- if eq .cloud "azure"}}
export AZURE_CLIENT_ID={{escape .AZURE_CLIENT_ID}}
export AZURE_CLIENT_SECRET={{escape .AZURE_CLIENT_SECRET}}
export AZURE_SUBSCRIPTION_ID={{escape .AZURE_SUBSCRIPTION_ID}}
export AZURE_TENANT_ID={{escape .AZURE_TENANT_ID}}
{{- end}}
{{- if eq .cloud "gcp"}}
export GOOGLE_APPLICATION_CREDENTIALS_DATA={{escape .GOOGLE_APPLICATION_CREDENTIALS_DATA}}
{{- end}}
{{- if eq .cloud "kubernetes"}}
export KUBERNETES_CONFIGURATION={{escape .KUBERNETES_CONFIGURATION}}
{{- end}}
{{- end}}

pip install virtualenv;
{{- if eq .dvc_ver "latest"}}
pip install dvc;
{{else}}
pip install dvc=={{.dvc_ver}};
{{- end}}
git clone {{.repo}} repo;
cd repo;
virtualenv -p python .env;
source .env/bin/activate;
pip install -r requirements.txt;
dvc exp run;
{{- if not .container}}
EOF
sudo chmod +x /usr/bin/dvc_run.sh

sudo bash -c 'cat << EOF > /etc/systemd/system/dvc-run.service
[Unit]
  After=default.target

[Service]
  Type=simple
  ExecStart=/usr/bin/dvc_run.sh

[Install]
  WantedBy=default.target
EOF'

{{- if .cloud}}
sudo systemctl daemon-reload
sudo systemctl enable dvc-run.service --now
{{- end}}
{{- end}}
`)
	var customDataBuffer bytes.Buffer
	err = tmpl.Execute(&customDataBuffer, data)

	if err == nil {
		script = customDataBuffer.String()
	}

	return script, err
}
