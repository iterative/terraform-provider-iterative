package destroy

import (
	"context"

	"github.com/spf13/cobra"

	cmdcommon "terraform-provider-iterative/cmd/common"
	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

type Options struct {
	BaseOptions cmdcommon.BaseOptions
	Workdir     string
	Output      string
}

func New() *cobra.Command {
	o := Options{}

	cmd := &cobra.Command{
		Use:   "destroy <name>",
		Short: "Destroy a task and all associated resources.",
		Long:  ``,
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			o.BaseOptions.ConfigureLogging()
			return nil
		},
		RunE: o.Run,
	}

	o.BaseOptions.SetFlags(cmd.Flags(), cmd)

	cmd.Flags().StringVar(&o.Output, "output", "", "output directory, relative to workdir")
	cmd.Flags().StringVar(&o.Workdir, "workdir", ".", "working directory")

	return cmd
}

func (o *Options) Run(cmd *cobra.Command, args []string) error {
	cloud := o.BaseOptions.GetCloud()
	cfg := common.Task{
		Environment: common.Environment{
			Directory:    o.Workdir,
			DirectoryOut: o.Output,
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
