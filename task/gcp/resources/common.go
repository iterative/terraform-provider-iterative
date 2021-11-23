package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func waitForOperation(ctx context.Context, timeout time.Duration, minimum time.Duration, maximum time.Duration, function func(...googleapi.CallOption) (*compute.Operation, error), arguments ...googleapi.CallOption) (*compute.Operation, error) {
	for delay, deadline := minimum, time.Now().Add(timeout); time.Now().Before(deadline); delay *= 2 {
		switch operation, err := function(arguments...); {
		case err == nil && operation.Status == "DONE":
			return operation, nil
		case err == nil && operation.Error != nil:
			return nil, fmt.Errorf("operation error: %#v", *operation.Error.Errors[0])
		case err != nil && !strings.HasSuffix(err.Error(), "resourceNotReady"):
			return nil, err
		}

		if delay < maximum {
			time.Sleep(delay)
		} else {
			time.Sleep(maximum)
		}
	}

	return nil, errors.New("timed out waiting for operation")
}
