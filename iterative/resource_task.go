package iterative

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aohorodnyk/uid"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

func resourceTask() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTaskCreate,
		DeleteContext: resourceTaskDelete,
		ReadContext:   resourceTaskRead,
		UpdateContext: resourceTaskRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"cloud": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"region": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "us-west",
			},
			"machine": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "m",
			},
			"disk_size": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Optional: true,
				Default:  30,
			},
			"spot": {
				Type:     schema.TypeFloat,
				ForceNew: true,
				Optional: true,
				Default:  -1,
			},
			"image": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "ubuntu",
			},
			"ssh_public_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"ssh_private_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"addresses": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"status": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"events": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"logs": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"script": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"workdir": {
				Optional: true,
				Type:     schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"input": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
							Default:  "",
						},
						"output": {
							Type:     schema.TypeString,
							ForceNew: false,
							Optional: true,
							Default:  "",
						},
					},
				},
			},
			"parallelism": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Optional: true,
				Default:  1,
			},
			"environment": {
				Type:     schema.TypeMap,
				ForceNew: true,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"timeout": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Optional: true,
				Default:  24 * time.Hour / time.Second,
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(15 * time.Minute),
			Read:   schema.DefaultTimeout(3 * time.Minute),
			Update: schema.DefaultTimeout(3 * time.Minute),
			Delete: schema.DefaultTimeout(15 * time.Minute),
		},
	}
}

func resourceTaskCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	task, err := resourceTaskBuild(ctx, d, m)
	if err != nil {
		return diagnostic(diags, err, diag.Error)
	}

	d.SetId(task.GetIdentifier(ctx).Long())

	if err := task.Create(ctx); err != nil {
		diags = diagnostic(diags, err, diag.Error)
		if err := task.Delete(ctx); err != nil {
			diags = diagnostic(diags, err, diag.Error)
		} else {
			diags = diagnostic(diags, errors.New("failed to create"), diag.Error)
			d.SetId("")
		}
	}

	return
}

func resourceTaskRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	task, err := resourceTaskBuild(ctx, d, m)
	if err != nil {
		return diagnostic(diags, err, diag.Warning)
	}

	if err := task.Read(ctx); err != nil {
		return diagnostic(diags, err, diag.Warning)
	}

	keyPair, err := task.GetKeyPair(ctx)
	if err != nil {
		return diagnostic(diags, err, diag.Warning)
	}

	publicKey, err := keyPair.PublicString()
	if err != nil {
		return diagnostic(diags, err, diag.Warning)
	}
	d.Set("ssh_public_key", publicKey)

	privateKey, err := keyPair.PrivateString()
	if err != nil {
		return diagnostic(diags, err, diag.Warning)
	}
	d.Set("ssh_private_key", privateKey)

	var addresses []string
	for _, address := range task.GetAddresses(ctx) {
		addresses = append(addresses, address.String())
	}
	d.Set("addresses", addresses)

	var events []string
	for _, event := range task.Events(ctx) {
		events = append(events, fmt.Sprintf(
			"%s: %s\n%s",
			event.Time.Format("2006-01-02 15:04:05"),
			event.Code,
			strings.Join(event.Description, "\n"),
		))
	}
	d.Set("events", events)

	status, err := task.Status(ctx)
	if err != nil {
		return diagnostic(diags, err, diag.Warning)
	}
	d.Set("status", status)

	logs, err := task.Logs(ctx)
	if err != nil {
		return diagnostic(diags, err, diag.Warning)
	}
	d.Set("logs", logs)

	d.SetId(task.GetIdentifier(ctx).Long())
	return diags
}

func resourceTaskDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	task, err := resourceTaskBuild(ctx, d, m)
	if err != nil {
		return diagnostic(diags, err, diag.Error)
	}

	if err := task.Delete(ctx); err != nil {
		return diagnostic(diags, err, diag.Error)
	}

	return
}

func resourceTaskBuild(ctx context.Context, d *schema.ResourceData, m interface{}) (task.Task, error) {
	v := make(map[string]*string)
	for name, value := range d.Get("environment").(map[string]interface{}) {
		v[name] = nil
		if contents := value.(string); contents != "" {
			v[name] = &contents
		}
	}

	val := "true"
	v["TPI_TASK"] = &val
	v["CI"] = nil
	v["CI_*"] = nil
	v["GITHUB_*"] = nil
	v["BITBUCKET_*"] = nil
	v["CML_*"] = nil
	v["REPO_TOKEN"] = nil

	c := common.Cloud{
		Provider: common.Provider(d.Get("cloud").(string)),
		Region:   common.Region(d.Get("region").(string)),
		Timeouts: common.Timeouts{
			Create: d.Timeout(schema.TimeoutCreate),
			Read:   d.Timeout(schema.TimeoutRead),
			Update: d.Timeout(schema.TimeoutUpdate),
			Delete: d.Timeout(schema.TimeoutDelete),
		},
	}

	directory := ""
	directory_out := ""
	if d.Get("workdir").(*schema.Set).Len() > 0 {
		storage := d.Get("workdir").(*schema.Set).List()[0].(map[string]interface{})
		directory = storage["input"].(string)

		directory_out = storage["output"].(string)
		if directory_out != "" && !isOutputValid(directory_out) {
			return nil, errors.New("output directory " + directory_out + " is not empty!")
		}
	}

	t := common.Task{
		Size: common.Size{
			Machine: d.Get("machine").(string),
			Storage: d.Get("disk_size").(int),
		},
		Environment: common.Environment{
			Image:        d.Get("image").(string),
			Script:       d.Get("script").(string),
			Variables:    v,
			Directory:    directory,
			DirectoryOut: directory_out,
			Timeout:      time.Duration(d.Get("timeout").(int)) * time.Second,
		},
		Firewall: common.Firewall{
			Ingress: common.FirewallRule{
				Ports: &[]uint16{22, 80}, // FIXME: just for testing Jupyter
			},
			// Egress is open on every port
		},
		Spot:        common.Spot(d.Get("spot").(float64)),
		Parallelism: uint16(d.Get("parallelism").(int)),
	}

	name := d.Id()
	if name == "" {
		if identifier := d.Get("name").(string); identifier != "" {
			name = identifier
		} else if identifier := os.Getenv("GITHUB_RUN_ID"); identifier != "" {
			name = identifier
		} else if identifier := os.Getenv("CI_PIPELINE_ID"); identifier != "" {
			name = identifier
		} else if identifier := os.Getenv("BITBUCKET_STEP_TRIGGERER_UUID"); identifier != "" {
			name = identifier
		} else {
			name = uid.NewProvider36Size(8).MustGenerate().String()
		}
	}

	return task.New(ctx, c, common.Identifier(name), t)
}

func diagnostic(diags diag.Diagnostics, err error, severity diag.Severity) diag.Diagnostics {
	return append(diags, diag.Diagnostic{
		Severity: severity,
		Summary:  err.Error(),
	})
}

func isOutputValid(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true
	}
	return false
}
