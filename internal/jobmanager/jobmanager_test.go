package jobmanager_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"terraform-provider-iterative/internal/jobmanager"
)

func TestJobManagerFailingJob(t *testing.T) {
	mgr := jobmanager.NewJobManager()

	job := &testJob{
		result: make(chan error),
	}

	// Test accessing the state of a non-existant job.
	_, err := mgr.GetStatus("no such job")
	assert.ErrorIs(t, err, jobmanager.ErrJobNotFound)

	id := mgr.Run(context.Background(), job)
	status, err := mgr.GetStatus(id)
	assert.NoError(t, err)
	assert.EqualValues(t, status, jobmanager.JobStatus{
		State: jobmanager.JobRunning,
	})

	job.result <- errors.New("some error")

	status, err = mgr.GetStatus(id)
	assert.NoError(t, err)
	assert.EqualValues(t, status, jobmanager.JobStatus{
		State: jobmanager.JobFailed,
		Error: "some error",
	})
}

func TestJobManagerSuccesfulJob(t *testing.T) {
	mgr := jobmanager.NewJobManager()

	job := &testJob{
		result: make(chan error),
	}

	id := mgr.Run(context.Background(), job)
	status, err := mgr.GetStatus(id)
	assert.NoError(t, err)
	assert.EqualValues(t, status, jobmanager.JobStatus{
		State: jobmanager.JobRunning,
	})

	job.result <- nil

	status, err = mgr.GetStatus(id)
	assert.NoError(t, err)
	assert.EqualValues(t, status, jobmanager.JobStatus{
		State: jobmanager.JobFinished,
	})
}

type testJob struct {
	result chan error
}

// Run implements server.Job.
func (t *testJob) Run(context.Context) error {
	return <-t.result
}
