package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"terraform-provider-iterative/task"
)

type listCmd struct {
	BaseOptions BaseOptions
}

func newListCmd() *cobra.Command {
	o := listCmd{}

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

func (o *listCmd) Run(cmd *cobra.Command, args []string) error {
	cloud := o.BaseOptions.GetCloud()
	ctx, cancel := context.WithTimeout(context.Background(), cloud.Timeouts.Read)
	defer cancel()

	lst, err := task.List(ctx, *cloud)
	if err != nil {
		return err
	}

	for _, id := range lst {
		fmt.Println(id.Long())
	}

	return nil
}
