package jobmanager

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"
)

var ErrJobNotFound = errors.New("job not found")

const JobIdLen = 8

func init() {
	rand.Seed(time.Now().UnixNano())
}

// JobState describes the execution state of a job.
type JobState string

const (
	JobRunning  JobState = "executing"
	JobFinished JobState = "done"
	JobFailed   JobState = "error"
)

// Job defines the interface for background jobs run by the job manager.
type Job interface {
	Run(ctx context.Context) error
}

// JobStatus defines the status of a job run by the job manager.
type JobStatus struct {
	State JobState
	Error string
}

// JobManager runs background jobs and provides access to their status.
type JobManager struct {
	m sync.RWMutex

	// TODO: implement cleanup of job statuses.
	jobStatus map[string]JobStatus
}

// NewJobManager creates a new job manager.
func NewJobManager() *JobManager {
	return &JobManager{
		jobStatus: make(map[string]JobStatus),
	}
}

// Run starts a background job and returns its id.
func (j *JobManager) Run(ctx context.Context, job Job) string {
	j.m.Lock()
	defer j.m.Unlock()

	id := randSeq(JobIdLen)

	// TODO: handle potential collisions.
	j.jobStatus[id] = JobStatus{State: JobRunning}

	go func(ctx context.Context) {
		err := job.Run(ctx)
		j.m.Lock()
		defer j.m.Unlock()
		if err == nil {
			j.jobStatus[id] = JobStatus{
				State: JobFinished,
			}
		} else {
			j.jobStatus[id] = JobStatus{
				State: JobFailed,
				Error: err.Error(),
			}
		}
	}(ctx)
	return id
}

// GetStatus returns the status of a job identified by its id.
// If an entry for the job does not exist, ErrJobNotFound will be returned.
func (j *JobManager) GetStatus(id string) (JobStatus, error) {
	j.m.RLock()
	defer j.m.RUnlock()

	status, ok := j.jobStatus[id]
	if !ok {
		return JobStatus{}, ErrJobNotFound
	}
	return status, nil
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
