package stop

import (
	"context"

	"github.com/spf13/cobra"

	cmdcommon "terraform-provider-iterative/cmd/common"
	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

type Options struct {
	BaseOptions cmdcommon.BaseOptions
}

func New() *cobra.Command {
	o := Options{}

	cmd := &cobra.Command{
		Use:    "stop <name>",
		Short:  "Stop a task, leaving supporting resources (e.g. storage) intact",
		Long:   ``,
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			o.BaseOptions.ConfigureLogging()
			return nil
		},
		RunE: o.Run,
	}
	o.BaseOptions.SetFlags(cmd.Flags(), cmd)

	return cmd
}

func (o *Options) Run(cmd *cobra.Command, args []string) error {
	cloud := o.BaseOptions.GetCloud()
	ctx, cancel := context.WithTimeout(context.Background(), cloud.Timeouts.Delete)
	defer cancel()

	id, err := common.ParseIdentifier(args[0])
	if err != nil {
		return err
	}

	tsk, err := task.New(ctx, *cloud, id, common.Task{})
	if err != nil {
		return err
	}

	if err := tsk.Stop(ctx); err != nil {
		return err
	}

	return nil
}
