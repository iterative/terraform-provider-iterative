package fakecs

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

//go:embed setup.sh
var SetupScript string

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
	o := Options{
		Environment: map[string]string{
			"GITHUB_PAT": "",
		},
		Image:         "ubuntu",
		Machine:       "t3.medium",
		Name:          "",
		Output:        "",
		Parallelism:   1,
		PermissionSet: "",
		Script:        SetupScript,
		Spot:          false,
		Storage:       -1,
		Timeout:       24 * 60 * 60,
		Workdir:       "",
	}

	cmd := &cobra.Command{
		Use:   "fakecs [repo...]",
		Short: "Create a not codespace",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd, args, cloud)
		},
	}

	return cmd
}

func (o *Options) Run(cmd *cobra.Command, args []string, cloud *common.Cloud) error {
	variables := make(map[string]*string)
	for name, value := range o.Environment {
		variables[name] = nil
		if value != "" {
			variables[name] = &value
		}
	}

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

	cfg.Spot = common.Spot(common.SpotDisabled)
	if o.Spot {
		cfg.Spot = common.Spot(common.SpotEnabled)
	}

	id := common.NewRandomIdentifier()

	if o.Name != "" {
		id = common.NewIdentifier(o.Name)
		if identifier, err := common.ParseIdentifier(o.Name); err == nil {
			id = identifier
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), cloud.Timeouts.Create)
	defer cancel()

	tsk, err := task.New(ctx, *cloud, id, cfg)
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
	//fmt.Println(o.Script)
	fmt.Println(id.Long())
	return nil
}
