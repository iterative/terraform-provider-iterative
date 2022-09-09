package list

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	cmdcommon "terraform-provider-iterative/cmd/common"
	"terraform-provider-iterative/task"
)

type Options struct {
	BaseOptions cmdcommon.BaseOptions
}

func New() *cobra.Command {
	o := Options{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		Long:  ``,
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
	ctx, cancel := context.WithTimeout(context.Background(), cloud.Timeouts.Read)
	defer cancel()

	lst, err := task.List(ctx, *cloud)
	if err != nil {
		return err
	}

	for _, id := range lst {
		logrus.Info(id.Long())
	}

	return nil
}
