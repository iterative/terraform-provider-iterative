package create

import (
	"context"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/aohorodnyk/uid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

type Options struct {
	Machine string
	Storage int
	Spot bool
	Image string
	Workdir string
	Output string
	Script string
	Environment map[string]string
	Timeout int
}

func New(cloud *common.Cloud) *cobra.Command {
	o := Options{}

	cmd := &cobra.Command{
		Use:   "create [command...]",
		Short: "Create a task",
		Long: ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd, args, cloud)
		},
	}

	cmd.Flags().StringVar(&o.Machine, "machine", "m", "machine type")
	cmd.Flags().IntVar(&o.Storage, "disk", -1, "disk size in gigabytes")
	cmd.Flags().BoolVar(&o.Spot, "spot", false, "use spot instances")
	cmd.Flags().StringVar(&o.Image, "image", "ubuntu", "machine image")
	cmd.Flags().StringVar(&o.Workdir, "workdir", ".", "working directory to upload")
	cmd.Flags().StringVar(&o.Output, "output", "", "output directory to download")
	cmd.Flags().StringVar(&o.Script, "script", "", "script to run")
	cmd.Flags().StringToStringVar(&o.Environment, "environment", map[string]string{}, "environment variables")
	cmd.Flags().IntVar(&o.Timeout, "timeout", 24 * 60 * 60, "timeout")

	return cmd
}

func (o *Options) Run(cmd *cobra.Command, args []string, cloud *common.Cloud) error {
	var variables map[string]*string
	for name, value := range o.Environment {
		variables[name] = nil
		if value != "" {
			variables[name] = &value
		}
	}

	script := o.Script
	if !strings.HasPrefix(script, "#!") {
		script = "#!/bin/sh\n"+script
	}
	script += "\n"+shellescape.QuoteCommand(args)

	cfg := common.Task{
		Size: common.Size{
			Machine: o.Machine,
			Storage: o.Storage,
		},
		Environment: common.Environment{
			Image:        o.Image,
			Script:       o.Script,
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
		Parallelism: uint16(1),
	}

	if o.Spot {
		cfg.Spot = common.Spot(common.SpotEnabled)
	} else {
		cfg.Spot = common.Spot(common.SpotDisabled)
	}

	name := common.Identifier(uid.NewProvider36Size(8).MustGenerate().String())

	ctx, cancel := context.WithTimeout(context.Background(), cloud.Timeouts.Create)
	defer cancel()

	tsk, err := task.New(ctx, *cloud, name, cfg)
	if err != nil {
		return err
	}

	if err := tsk.Create(ctx); err != nil {
		logrus.Errorf("Failed to create a new task: %v", err)
		logrus.Warn("Attempting to delete residual resources...")
		if err := tsk.Delete(ctx); err != nil {
			logrus.Errorf("Failed to delete residual resources")
			return err
		}
		return err
	}

	logrus.Info(name)
	return nil
}