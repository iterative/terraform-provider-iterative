package fakecs

import (
	"context"
	_ "embed"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

//go:embed setup.sh
var SetupScript string

type Options struct {
	Environment   map[string]string
	Image         string
	Machine       string
	Name          string
	Output        string
	Parallelism   int
	PermissionSet string
	Script        string
	Spot          bool
	Storage       int
	Tags          map[string]string
	Timeout       int
	Workdir       string
	Repo          string
}

func New(cloud *common.Cloud) *cobra.Command {
	o := Options{
		Environment: map[string]string{
			"GITHUB_PAT": "",
		},
		Image:         "ubuntu",
		Machine:       "t3.medium",
		Name:          "",
		Output:        "",
		Parallelism:   1,
		PermissionSet: "arn:aws:iam::342840881361:instance-profile/tpi-vscode-example",
		Script:        SetupScript,
		Spot:          false,
		Storage:       -1,
		Timeout:       24 * 60 * 60,
		Workdir:       "",
	}

	cmd := &cobra.Command{
		Use:   "fakecs [repo...]",
		Short: "Create a not codespace",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd, args, cloud)
		},
	}
	cmd.Flags().StringVar(&o.Repo, "repo", "iterative/cml", "GitHub repo to clone")

	return cmd
}

func (o *Options) Run(cmd *cobra.Command, args []string, cloud *common.Cloud) error {
	variables := make(map[string]*string)
	for name, value := range o.Environment {
		variables[name] = nil
		if value != "" {
			variables[name] = &value
		}
	}
	gitOrg := strings.Split(o.Repo, "/")[0]
	gitRepo := strings.Split(o.Repo, "/")[1]
	script := strings.ReplaceAll(strings.ReplaceAll(SetupScript, "GIT_ORG", gitOrg), "GIT_REPO", gitRepo)
	cfg := common.Task{
		Size: common.Size{
			Machine: o.Machine,
			Storage: o.Storage,
		},
		Environment: common.Environment{
			Image:        o.Image,
			Script:       script,
			Variables:    variables,
			Directory:    o.Workdir,
			DirectoryOut: o.Output,
			Timeout:      time.Duration(o.Timeout) * time.Second,
		},
		Firewall: common.Firewall{
			Ingress: common.FirewallRule{
				Ports: &[]uint16{22},
			},
		},
		Parallelism:   uint16(1),
		PermissionSet: o.PermissionSet,
	}

	cfg.Spot = common.Spot(common.SpotDisabled)
	if o.Spot {
		cfg.Spot = common.Spot(common.SpotEnabled)
	}

	id := common.NewRandomIdentifier()

	if o.Name != "" {
		id = common.NewIdentifier(o.Name)
		if identifier, err := common.ParseIdentifier(o.Name); err == nil {
			id = identifier
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), cloud.Timeouts.Create)
	defer cancel()

	tsk, err := task.New(ctx, *cloud, id, cfg)
	if err != nil {
		return err
	}

	if err := tsk.Create(ctx); err != nil {
		logrus.Errorf("Failed to create a new task: %v", err)
		logrus.Warn("Attempting to delete residual resources...")
		if err := tsk.Delete(ctx); err != nil {
			logrus.Errorf("Failed to delete residual resources")
			return err
		}
		return err
	}
	fmt.Println(id.Long())
	for i := 0; i < 5; i++ {
		logrus.Info("waiting 30s")
		time.Sleep(30 * time.Second)
		if err := tsk.Read(ctx); err != nil {
			logrus.Warn("Failed to read task")
			return err
		}
		logs, _ := tsk.Logs(ctx)
		for _, log := range logs {
			for _, line := range strings.Split(strings.Trim(log, "\n"), "\n") {
				temp := line[20:]
				if temp == "***READY***" {
					// exec code --remote ssh-remote+ubuntu@${iterative_task.vscode.addresses[0]} /home/ubuntu/magnetic-tiles-defect"
					fmt.Println("running: ", fmt.Sprintf("code --remote ssh-remote+ubuntu@%s /home/ubuntu/%s", tsk.GetAddresses(ctx)[0], gitRepo))
					cmd := exec.Command("code", "--remote", fmt.Sprintf("ssh-remote+ubuntu@%s", tsk.GetAddresses(ctx)[0]), fmt.Sprintf("/home/ubuntu/%s", gitRepo))
					err := cmd.Run()
					fmt.Println(id.Long())
					return err
				}
			}
		}
		fmt.Println("not ready")
	}
	return nil
}
