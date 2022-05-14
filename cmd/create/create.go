package create

import (
	"context"
	"strings"
	"time"

	"github.com/aohorodnyk/uid"
	"github.com/spf13/cobra"
	"github.com/sirupsen/logrus"

	"gopkg.in/alessio/shellescape.v1"

	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

var (
	machine string
	storage int
	spot bool
	image string
	workdir string
	output string
	environment map[string]string
	timeout int
)

func New(cloud *common.Cloud) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [command...]",
		Short: "Create a task",
		Long: ``,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, cloud)
		},
	}

	cmd.Flags().StringVar(&machine, "machine", "m", "machine type")
	cmd.Flags().IntVar(&storage, "disk", -1, "disk size in gigabytes")
	cmd.Flags().BoolVar(&spot, "spot", false, "use spot instances")
	cmd.Flags().StringVar(&image, "image", "ubuntu", "machine image")
	cmd.Flags().StringVar(&workdir, "workdir", ".", "working directory to upload")
	cmd.Flags().StringVar(&output, "output", "", "output directory to download")
	cmd.Flags().StringToStringVar(&environment, "environment", map[string]string{}, "environment variables")
	cmd.Flags().IntVar(&timeout, "timeout", 24 * 60 * 60, "timeout")

	return cmd
}

func run(cmd *cobra.Command, args []string, cloud *common.Cloud) error {
	logrus.Info(environment)
	variables := make(map[string]*string)
	for name, value := range environment {
		variables[name] = nil
		if value != "" {
			variables[name] = &value
		}
	}

	script := strings.Join(args, " ")
	if !strings.HasPrefix(script, "#!") {
		script = "#!/bin/sh\n"
		for _, arg := range args {
			script += shellescape.Quote(arg) + " "
		}
	}

	cfg := common.Task{
		Size: common.Size{
			Machine: machine,
			Storage: storage,
		},
		Environment: common.Environment{
			Image:        image,
			Script:       script,
			Variables:    variables,
			Directory:    workdir,
			DirectoryOut: output,
			Timeout:      time.Duration(timeout) * time.Second,
		},
		Firewall: common.Firewall{
			Ingress: common.FirewallRule{
				Ports: &[]uint16{22},
			},
		},
		Parallelism: uint16(1),
	}

	if spot {
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