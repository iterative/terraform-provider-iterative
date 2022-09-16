package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

type statusCmd struct {
	BaseOptions BaseOptions

	Parallelism int
	Timestamps  bool
	Status      bool
	Logs        bool
}

func newStatusCmd() *cobra.Command {
	o := statusCmd{}

	cmd := &cobra.Command{
		Use:   "status <name>",
		Short: "Get the status of a task",
		Long:  ``,
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			o.BaseOptions.ConfigureLogging()
			return nil
		},
		RunE: o.Run,
	}

	o.BaseOptions.SetFlags(cmd.Flags())

	cmd.Flags().IntVar(&o.Parallelism, "parallelism", 1, "parallelism")
	cmd.Flags().BoolVar(&o.Timestamps, "timestamps", false, "display timestamps")
	cmd.Flags().BoolVar(&o.Status, "status", true, "read status")
	cmd.Flags().BoolVar(&o.Logs, "logs", false, "read logs")
	cmd.MarkFlagsMutuallyExclusive("status", "logs")

	return cmd
}

func (o *statusCmd) Run(cmd *cobra.Command, args []string) error {
	cloud := o.BaseOptions.GetCloud()
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
	case o.Logs:
		return o.printLogs(ctx, tsk)
	case o.Status:
		return o.printStatus(ctx, tsk)
	}

	return nil
}

func (o *statusCmd) printLogs(ctx context.Context, tsk task.Task) error {
	logs, err := tsk.Logs(ctx)
	if err != nil {
		return err
	}

	for _, log := range logs {
		for _, line := range strings.Split(strings.Trim(log, "\n"), "\n") {
			if !o.Timestamps {
				_, line, _ = strings.Cut(line, " ")
			}
			fmt.Println(line)
		}
	}

	return nil
}

func (o *statusCmd) printStatus(ctx context.Context, tsk task.Task) error {
	for _, event := range tsk.Events(ctx) {
		line := fmt.Sprintf("%s: %s", event.Code, strings.Join(event.Description, " "))
		if o.Timestamps {
			line = fmt.Sprintf("%s %s", event.Time.Format("2006-01-02T15:04:05Z"), line)
		}

		logrus.Debug(line)
	}

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
	return nil
}
