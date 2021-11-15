package iterative

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/universal"
)

func resourceTask() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTaskCreate,
		DeleteContext: resourceTaskDelete,
		ReadContext:   resourceTaskRead,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"cloud": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"region": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "us-west",
			},
			"machine": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "m",
			},
			"disk_size": &schema.Schema{
				Type:     schema.TypeInt,
				ForceNew: true,
				Optional: true,
				Default:  30,
			},
			"spot": &schema.Schema{
				Type:     schema.TypeFloat,
				ForceNew: true,
				Optional: true,
				Default:  -1,
			},
			"image": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "ubuntu",
			},
			"ssh_public_key": &schema.Schema{
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"ssh_private_key": &schema.Schema{
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"addresses": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"status": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"events": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"logs": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"script": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"directory": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
			"parallelism": &schema.Schema{
				Type:     schema.TypeInt,
				ForceNew: true,
				Optional: true,
				Default:  1,
			},
			"environment": &schema.Schema{
				Type:     schema.TypeMap,
				ForceNew: true,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"timeout": &schema.Schema{
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

	if err := task.Create(ctx); err != nil {
		return diagnostic(diags, err, diag.Error)
	}

	d.SetId(task.GetIdentifier(ctx))
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
	for _, event := range task.GetEvents(ctx) {
		events = append(events, fmt.Sprintf(
			"%s: %s\n%s",
			event.Time.Format("2006-01-02 15:04:05"),
			event.Code,
			strings.Join(event.Description, "\n"),
		))
	}
	d.Set("events", events)

	d.Set("status", task.GetStatus(ctx))

	logs, err := task.Logs(ctx)
	if err != nil {
		return diagnostic(diags, err, diag.Warning)
	}
	d.Set("logs", logs)

	d.SetId(task.GetIdentifier(ctx))
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

	d.SetId("")
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

	c := universal.Cloud{
		Provider: universal.Provider(d.Get("cloud").(string)),
		Region:   universal.Region(d.Get("region").(string)),
		Timeouts: universal.Timeouts{
			Create: d.Timeout(schema.TimeoutCreate),
			Read:   d.Timeout(schema.TimeoutRead),
			Update: d.Timeout(schema.TimeoutUpdate),
			Delete: d.Timeout(schema.TimeoutDelete),
		},
	}

	t := universal.Task{
		Size: universal.Size{
			Machine: d.Get("machine").(string),
			Storage: d.Get("disk_size").(int),
		},
		Environment: universal.Environment{
			Image:     d.Get("image").(string),
			Script:    d.Get("script").(string),
			Variables: v,
			Directory: d.Get("directory").(string),
			Timeout:   time.Duration(d.Get("timeout").(int)) * time.Second,
		},
		Firewall: universal.Firewall{
			Ingress: universal.FirewallRule{
				Ports: &[]uint16{22, 80}, // FIXME: just for testing Jupyter
			},
			// Egress is open on every port
		},
		Spot:        d.Get("spot").(float64),
		Parallelism: uint16(d.Get("parallelism").(int)),
	}

	return task.NewTask(ctx, c, d.Get("name").(string), t)
}

func diagnostic(diags diag.Diagnostics, err error, severity diag.Severity) diag.Diagnostics {
	return append(diags, diag.Diagnostic{
		Severity: severity,
		Summary:  err.Error(),
	})
}
