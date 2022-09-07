package read

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

type Options struct {
	Parallelism int
	Timestamps  bool
	Status      bool
	Events      bool
	Logs        bool
}

func New(cloud *common.Cloud) *cobra.Command {
	o := Options{}

	cmd := &cobra.Command{
		Use:   "read <name>",
		Short: "Read information from an existing task",
		Long:  ``,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd, args, cloud)
		},
	}

	cmd.Flags().IntVar(&o.Parallelism, "parallelism", 1, "parallelism")
	cmd.Flags().BoolVar(&o.Timestamps, "timestamps", false, "display timestamps")
	cmd.Flags().BoolVar(&o.Status, "status", true, "read status")
	cmd.Flags().BoolVar(&o.Logs, "logs", false, "read logs")
	cmd.MarkFlagsMutuallyExclusive("status", "logs")

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
	case o.Logs:
		return o.printLogs(ctx, tsk)
	case o.Status:
		return o.printStatus(ctx, tsk)
	}

	return nil
}

func (o *Options) printLogs(ctx context.Context, tsk task.Task) error {
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

func (o *Options) printStatus(ctx context.Context, tsk task.Task) error {
	for _, event := range tsk.Events(ctx) {
		line := fmt.Sprintf("%s: %s", event.Code, strings.Join(event.Description, " "))
		if o.Timestamps {
			line = fmt.Sprintf("%s %s", event.Time.Format("2006-01-02T15:04:05Z"), line)
		}

		logrus.Info(line)
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
