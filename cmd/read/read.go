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
	Parallelism   int
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

	var events []string
	for _, event := range tsk.Events(ctx) {
		events = append(events, fmt.Sprintf(
			"%s: %s\n%s",
			event.Time.Format("2006-01-02 15:04:05"),
			event.Code,
			strings.Join(event.Description, "\n"),
		))
	}
	logrus.Info(events)

	logs, err := tsk.Logs(ctx)
	if err != nil {
		return err
	}

	for index, log := range logs {
		for _, line := range strings.Split(strings.Trim(log, "\n"), "\n") {
			logrus.Infof("\x1b[%dmLOG %d >> %s", 35, index, line)
		}
	}

	status, err := tsk.Status(ctx)
	if err != nil {
		return err
	}

	message := fmt.Sprintf("\x1b[%dmStatus: queued \x1b[1m•\x1b[0m", 34)

	if status["succeeded"] >= o.Parallelism {
		message = fmt.Sprintf("\x1b[%dmStatus: completed successfully \x1b[1m•\x1b[0m", 32)
	}
	if status["failed"] > 0 {
		message = fmt.Sprintf("\x1b[%dmStatus: completed with errors \x1b[1m•\x1b[0m", 31)
	}
	if status["running"] >= o.Parallelism {
		message = fmt.Sprintf("\x1b[%dmStatus: running \x1b[1m•\x1b[0m", 33)
	}

	logrus.Info(message)
	return nil
}