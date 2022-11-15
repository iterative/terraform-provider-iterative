package server

import (
	"context"
	"log"
	"time"

	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

// createTaskJob runs creation of cloud resources for a specific task.
type createTaskJob struct {
	task  common.Task
	cloud common.Cloud
	id    common.Identifier
}

func newCreateTaskJob(taskParams Task, credentials *CloudCredentials) *createTaskJob {
	var envVars = common.Variables{}
	if taskParams.Env != nil {
		for key, value := range *taskParams.Env {
			envValue := value
			envVars[key] = &envValue
		}
	}
	spot := common.SpotDisabled
	if taskParams.Spot != nil && *taskParams.Spot {
		spot = common.SpotEnabled
	}
	if taskParams.Storage == 0 {
		taskParams.Storage = -1
	}
	taskConfig := common.Task{
		Size: common.Size{
			Machine: taskParams.Machine,
			Storage: taskParams.Storage,
		},
		Environment: common.Environment{
			Image:        taskParams.Image,
			Script:       taskParams.Script,
			Variables:    envVars,
			Directory:    "",
			DirectoryOut: "",
			ExcludeList:  nil,
			Timeout:      time.Duration(taskParams.Timeout) * time.Second,
		},
		Firewall: common.Firewall{
			Ingress: common.FirewallRule{
				Ports: &[]uint16{22},
			},
		},
		Parallelism: uint16(1),
		Spot:        spot,
	}

	cloud := common.Cloud{
		Provider:    credentials.Provider,
		Region:      credentials.Region,
		Credentials: credentials.GetCredentials(),
		Timeouts: common.Timeouts{
			Create: 15 * time.Minute,
			Read:   3 * time.Minute,
			Update: 3 * time.Minute,
			Delete: 15 * time.Minute,
		},
	}
	id := common.NewRandomIdentifier("")
	log.Printf("creating task %q", id.Long())
	return &createTaskJob{
		task:  taskConfig,
		cloud: cloud,
		id:    id,
	}
}

// Run implements the jobmanager.Job interface.
func (j *createTaskJob) Run(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, j.cloud.Timeouts.Create)
	defer cancel()

	tsk, err := task.New(ctx, j.cloud, j.id, j.task)
	if err != nil {
		return err
	}

	err = tsk.Create(ctx)
	if err != nil {
		log.Printf("Failed to create a new task: %v", err)
		if err := tsk.Delete(ctx); err != nil {
			log.Printf("Failed to delete residual resources")
			return err
		}
		return err
	}
	return nil
}

// destroyTaskJob deallocates cloud resources associated with a task.
type destroyTaskJob struct {
	task  common.Task
	cloud common.Cloud
	id    common.Identifier
}

func newDestroyTaskJob(taskId string, credentials *CloudCredentials) (*destroyTaskJob, error) {

	id, err := common.ParseIdentifier(taskId)
	if err != nil {
		return nil, err
	}
	cloud := common.Cloud{
		Provider:    credentials.Provider,
		Region:      credentials.Region,
		Credentials: credentials.GetCredentials(),
		Timeouts: common.Timeouts{
			Create: 15 * time.Minute,
			Read:   3 * time.Minute,
			Update: 3 * time.Minute,
			Delete: 15 * time.Minute,
		},
	}
	task := common.Task{
		Environment: common.Environment{
			Image: "ubuntu",
		},
	}

	return &destroyTaskJob{
		id:    id,
		cloud: cloud,
		task:  task,
	}, nil
}

// Run implements the jobmanager.Job interface.
func (j *destroyTaskJob) Run(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, j.cloud.Timeouts.Read)
	defer cancel()

	tsk, err := task.New(ctx, j.cloud, j.id, j.task)
	if err != nil {
		return err
	}
	return tsk.Delete(ctx)
}
