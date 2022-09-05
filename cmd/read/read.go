package read

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

type Options struct {
	Parallelism int
	Status      bool
	Events      bool
	Logs        bool
}

func New(cloud *common.Cloud) *cobra.Command {
	o := Options{}

	cmd := &cobra.Command{
		Use:   "read <name>",
		Short: "Read the status of a task",
		Long:  ``,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd, args, cloud)
		},
	}

	cmd.Flags().IntVar(&o.Parallelism, "parallelism", 1, "parallelism")
	cmd.Flags().BoolVar(&o.Status, "status", false, "Read status")
	cmd.Flags().BoolVar(&o.Events, "events", false, "Read events")
	cmd.Flags().BoolVar(&o.Logs, "logs", false, "Read logs")
	cmd.MarkFlagsMutuallyExclusive("status", "events", "logs")

	return cmd
}

func (o *Options) Run(cmd *cobra.Command, args []string, cloud *common.Cloud) error {
	cfg := common.Task{
		Environment: common.Environment{
			Image: "ubuntu",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), cloud.Timeouts.Read)
	defer cancel()

	id, err := common.ParseIdentifier(args[0])
	if err != nil {
		return err
	}

	tsk, err := task.New(ctx, *cloud, id, cfg)
	if err != nil {
		return err
	}

	if err := tsk.Read(ctx); err != nil {
		return err
	}

	switch {
	case o.Events:
		var events []string
		for _, event := range tsk.Events(ctx) {
			events = append(events, fmt.Sprintf(
				"%s: %s\n%s",
				event.Time.Format("2006-01-02 15:04:05"),
				event.Code,
				strings.Join(event.Description, "\n"),
			))
		}

		fmt.Println(events)
	case o.Logs:
		logs, err := tsk.Logs(ctx)
		if err != nil {
			return err
		}

		for _, log := range logs {
			for _, line := range strings.Split(strings.Trim(log, "\n"), "\n") {
				fmt.Println(line)
			}
		}
	case o.Status:
		status, err := tsk.Status(ctx)
		if err != nil {
			return err
		}

		message := "queued"

		if status["succeeded"] >= o.Parallelism {
			message = "succeeded"
		}
		if status["failed"] > 0 {
			message = "failed"
		}
		if status["running"] >= o.Parallelism {
			message = "running"
		}

		fmt.Println(message)
	}

	return nil
}
