// Package server implements a http router that handles creating, listing
// and destroying cloud resources.
package server

// TODO: logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"terraform-provider-iterative/internal/jobmanager"
	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

type server struct {
	ServerInterface

	jobManager *jobmanager.JobManager
}

// NewServer creates a new server instance.
func NewServer() *server {
	return &server{
		jobManager: jobmanager.NewJobManager(),
	}
}

// CreateTask allocates cloud resources for executing a task.
func (s *server) CreateTask(w http.ResponseWriter, r *http.Request) {
	// Read credentials.
	creds, err := CredentialsFromRequest(r)
	if err != nil {
		RespondError(r.Context(), w, err)
		return
	}

	defer r.Body.Close()
	var taskParams Task
	err = json.NewDecoder(r.Body).Decode(&taskParams)
	if err != nil {
		RespondError(r.Context(), w, err)
		return
	}

	job := newCreateTaskJob(taskParams, creds)
	jobId := s.jobManager.Run(context.Background(), job)

	response := Job{
		Id: jobId,
	}
	RespondValue(r.Context(), w, response)
}

// DestroyTask deallocates cloud resources associated with a task.
func (s *server) DestroyTask(w http.ResponseWriter, r *http.Request, id string) {
	creds, err := CredentialsFromRequest(r)
	if err != nil {
		RespondError(r.Context(), w, err)
		return
	}
	job, err := newDestroyTaskJob(id, creds)
	if err != nil {
		RespondError(r.Context(), w, err)
		return
	}
	jobId := s.jobManager.Run(context.Background(), job)

	response := Job{
		Id: jobId,
	}
	RespondValue(r.Context(), w, response)
}

func (s *server) ListTasks(w http.ResponseWriter, r *http.Request) {
	// Read credentials.
	creds, err := CredentialsFromRequest(r)
	if err != nil {
		RespondError(r.Context(), w, err)
		return
	}
	cloud := common.Cloud{
		Provider:    creds.Provider,
		Region:      common.DefaultRegion,
		Credentials: creds.GetCredentials(),
		Timeouts: common.Timeouts{
			Create: 15 * time.Minute,
			Read:   3 * time.Minute,
			Update: 3 * time.Minute,
			Delete: 15 * time.Minute,
		},
	}
	lst, err := task.List(r.Context(), cloud)
	if err != nil {
		log.Printf("failed to list tasks: %v", err)
		RespondError(r.Context(), w, err)
		return
	}
	result := make([]string, len(lst))
	for i, id := range lst {
		result[i] = id.Long()
	}
	response := TaskList{
		Tasks: &result,
	}
	RespondValue(r.Context(), w, response)
}

func (s *server) GetTaskStatus(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	creds, err := CredentialsFromRequest(r)
	if err != nil {
		RespondError(ctx, w, err)
		return
	}
	cloud := common.Cloud{
		Provider:    creds.Provider,
		Region:      common.DefaultRegion,
		Credentials: creds.GetCredentials(),
		Timeouts: common.Timeouts{
			Create: 15 * time.Minute,
			Read:   3 * time.Minute,
			Update: 3 * time.Minute,
			Delete: 15 * time.Minute,
		},
	}
	taskParams := common.Task{
		Environment: common.Environment{
			Image: "ubuntu",
		},
	}
	taskId, err := common.ParseIdentifier(id)
	if err != nil {
		RespondError(ctx, w, err)
		return
	}

	tsk, err := task.New(ctx, cloud, taskId, taskParams)
	if err != nil {
		RespondError(ctx, w, err)
		return
	}
	status, err := tsk.Status(ctx)
	if err != nil {
		RespondError(ctx, w, err)
		return
	}
	var response TaskStatus
	for code, count := range status {
		if count == 0 {
			// We're only expecting 1 non-zero status here.
			continue
		}
		switch code {
		case common.StatusCodeActive:
			response.Status = Active
		case common.StatusCodeSucceeded:
			response.Status = Succeeded
		case common.StatusCodeFailed:
			response.Status = Failed
		}
	}
	// Add log output.
	logs, err := tsk.Logs(ctx)
	if err != nil {
		RespondError(ctx, w, err)
		return
	}
	response.Logs = logs

	// Add event data.
	events := tsk.Events(ctx)
	if len(events) > 0 {
		response.Events = make([]string, len(events))
		for i, event := range tsk.Events(ctx) {
			evtLine := fmt.Sprintf("%s (%s) %s", event.Time.Format(time.RFC3339), event.Code, event.Description)
			response.Events[i] = evtLine
		}
	}
	RespondValue(ctx, w, response)
}

// GetJobStatus implements ServerInterface.GetJobStatus.
func (s *server) GetJobStatus(w http.ResponseWriter, r *http.Request, id string) {
	state, err := s.jobManager.GetStatus(id)
	if err == jobmanager.ErrJobNotFound {
		log.Printf("job id %q not found", id)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	jobState := JobStatus(state.State)
	var jobError *string
	if state.Error != "" {
		stateError := state.Error
		jobError = &stateError
	}
	response := Job{
		Id:     id,
		Status: &jobState,
		Error:  jobError,
	}
	RespondValue(r.Context(), w, response)
}

// GetKey implements ServerInterface.GetKey.
func (s *server) GetKey(w http.ResponseWriter, r *http.Request) {
	key := PublicKey[:]
	RespondValue(r.Context(), w, EncryptionKey{
		Key: &key,
	})
}

// RespondValue writes the provided object (marshalled to json) to the response.
func RespondValue(ctx context.Context, w http.ResponseWriter, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(value)
	if err != nil {
		log.Printf("failed to marshal response: %s", err.Error())
	}
}

// Interface implementation validation.
var _ ServerInterface = (*server)(nil)
