package delete

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/iterative/terraform-provider-iterative/task"
	"github.com/iterative/terraform-provider-iterative/task/common"
)

type Options struct {
	Workdir     string
	Output      string
	ExcludeList []string
}

func New(cloud *common.Cloud) *cobra.Command {
	o := Options{}

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a task",
		Long:  ``,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd, args, cloud)
		},
	}

	cmd.Flags().StringVar(&o.Output, "output", "", "output directory, relative to workdir")
	cmd.Flags().StringVar(&o.Workdir, "workdir", ".", "working directory")

	return cmd
}

func (o *Options) Run(cmd *cobra.Command, args []string, cloud *common.Cloud) error {
	cfg := common.Task{
		Environment: common.Environment{
			Directory:    o.Workdir,
			DirectoryOut: o.Output,
			ExcludeList:  o.ExcludeList,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), cloud.Timeouts.Delete)
	defer cancel()

	id, err := common.ParseIdentifier(args[0])
	if err != nil {
		return err
	}

	tsk, err := task.New(ctx, *cloud, id, cfg)
	if err != nil {
		return err
	}
	return tsk.Delete(ctx)
}
