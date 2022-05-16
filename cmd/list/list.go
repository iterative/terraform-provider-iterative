package list

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/sirupsen/logrus"

	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

func New(cloud *common.Cloud) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		Long: ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, cloud)
		},
	}

	return cmd
}


func run(cmd *cobra.Command, args []string, cloud *common.Cloud) error {
	ctx, cancel := context.WithTimeout(context.Background(), cloud.Timeouts.Read)
	defer cancel()

	lst, err := task.List(ctx, *cloud)
	if err != nil {
		return err
	}

	for _, id := range lst {
		logrus.Info(id)
	}

	return nil
}