package create

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

type Options struct {
	Environment   map[string]string
	Image         string
	Machine       string
	Name          string
	Output        string
	Parallelism   int
	PermissionSet string
	Script        string
	Spot          bool
	Storage       int
	Tags          map[string]string
	Timeout       int
	Workdir       string
}

func New(cloud *common.Cloud) *cobra.Command {
	o := Options{}

	cmd := &cobra.Command{
		Use:   "create [command...]",
		Short: "Create a task",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd, args, cloud)
		},
	}

	cmd.Flags().StringToStringVar(&o.Environment, "environment", map[string]string{}, "environment variables")
	cmd.Flags().StringVar(&o.Image, "image", "ubuntu", "machine image")
	cmd.Flags().StringVar(&o.Machine, "machine", "m", "machine type")
	cmd.Flags().StringVar(&o.Name, "name", "", "deterministic name")
	cmd.Flags().StringVar(&o.Output, "output", "", "output directory to download")
	cmd.Flags().IntVar(&o.Parallelism, "parallelism", 1, "parallelism")
	cmd.Flags().StringVar(&o.PermissionSet, "permission-set", "", "permission set")
	cmd.Flags().StringVar(&o.Script, "script", "", "script to run")
	cmd.Flags().BoolVar(&o.Spot, "spot", false, "use spot instances")
	cmd.Flags().IntVar(&o.Storage, "disk-size", -1, "disk size in gigabytes")
	cmd.Flags().StringToStringVar(&o.Tags, "tags", map[string]string{}, "resource tags")
	cmd.Flags().IntVar(&o.Timeout, "timeout", 24*60*60, "timeout")
	cmd.Flags().StringVar(&o.Workdir, "workdir", ".", "working directory to upload")

	return cmd
}

func (o *Options) Run(cmd *cobra.Command, args []string, cloud *common.Cloud) error {
	variables := make(map[string]*string)
	for name, value := range o.Environment {
		name = strings.ToUpper(name)
		variables[name] = nil
		if copy := value; value != "" {
			variables[name] = &copy
		}
	}

	script := o.Script
	if !strings.HasPrefix(script, "#!") {
		script = "#!/bin/sh\n" + script
	}
	script += "\n" + shellescape.QuoteCommand(args)

	cfg := common.Task{
		Size: common.Size{
			Machine: o.Machine,
			Storage: o.Storage,
		},
		Environment: common.Environment{
			Image:        o.Image,
			Script:       script,
			Variables:    variables,
			Directory:    o.Workdir,
			DirectoryOut: o.Output,
			Timeout:      time.Duration(o.Timeout) * time.Second,
		},
		Firewall: common.Firewall{
			Ingress: common.FirewallRule{
				Ports: &[]uint16{22},
			},
		},
		Parallelism:   uint16(1),
		PermissionSet: o.PermissionSet,
	}

	cfg.Spot = common.Spot(common.SpotDisabled)
	if o.Spot {
		cfg.Spot = common.Spot(common.SpotEnabled)
	}

	id := common.NewRandomIdentifier(o.Name)
	if identifier, err := common.ParseIdentifier(o.Name); err == nil {
		id = identifier
	}

	ctx, cancel := context.WithTimeout(context.Background(), cloud.Timeouts.Create)
	defer cancel()

	tsk, err := task.New(ctx, *cloud, id, cfg)
	if err != nil {
		return err
	}

	fmt.Println(id.Long())
	
	if err := tsk.Create(ctx); err != nil {
		logrus.Errorf("Failed to create a new task: %v", err)
		logrus.Warn("Attempting to delete residual resources...")
		if err := tsk.Delete(ctx); err != nil {
			logrus.Errorf("Failed to delete residual resources")
			return err
		}
		return err
	}

	return nil
}
