package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/0x2b3bfa0/logrusctx"
	"github.com/sirupsen/logrus"

	"terraform-provider-iterative/task"
	"terraform-provider-iterative/task/common"
)

type Operation string

const (
	OperationCreate = "create"
	OperationRead   = "read"
	OperationDelete = "delete"
	OperationList   = "list"
)

type Command struct {
	Operation Operation

	Request string
	Name    string

	Cloud common.Cloud
	Task  common.Task
}

func main() {
	lambda.Start(forking(context.Background(), handler))
}

func handler(ctx context.Context, req *events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	initAPIGatewayManagementAPIClient(req) // HACK

	logger := logrus.New()
	logger.SetOutput(WebSocketWriter{ConnnectionID: req.RequestContext.ConnectionID})
	logger.SetFormatter(&logrus.JSONFormatter{})

	body := []byte(req.Body)

	if req.IsBase64Encoded {
		if decoded, err := base64.StdEncoding.DecodeString(req.Body); err == nil {
			body = decoded
		} else {
			logger.WithError(err).Error("invalid Base64-encoded request")
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}
	}

	command := Command{
		Cloud: common.Cloud{
			Timeouts: common.Timeouts{
				Create: 14 * time.Minute,
				Read:   14 * time.Minute,
				Update: 14 * time.Minute,
				Delete: 14 * time.Minute,
			},
		},
		Task: common.Task{
			Environment: common.Environment{
				Timeout: 24 * time.Hour,
			},
		},
	}

	if err := json.Unmarshal(body, &command); err != nil {
		logger.WithError(err).Error("invalid command payload")
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	if command.Request == "" {
		command.Request = req.RequestContext.RequestID
	}

	ctx = logrusctx.WithLogger(ctx, logger.WithField("request", command.Request))

	var err error
	switch command.Operation {
	case OperationList:
		err = list(ctx, command)
	case OperationCreate:
		err = create(ctx, command)
	case OperationDelete:
		err = delete(ctx, command)
	case OperationRead:
		err = read(ctx, command)
	default:
		err = fmt.Errorf("invalid operation: %s", command.Operation)
	}

	if err != nil {
		logrusctx.Logger(ctx).WithError(err).Error("operation failed")
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	logrusctx.Info(ctx, "operation succeeded")
	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
}

func list(ctx context.Context, cmd Command) error {
	tasks, err := task.List(ctx, cmd.Cloud)
	if err != nil {
		return err
	}

	for _, t := range tasks {
		logrusctx.Info(ctx, t.Long())
	}

	return nil
}

func create(ctx context.Context, cmd Command) error {
	id := common.NewRandomIdentifier(cmd.Name)
	if identifier, err := common.ParseIdentifier(cmd.Name); err == nil {
		id = identifier
	}

	tsk, err := task.New(ctx, cmd.Cloud, id, cmd.Task)
	if err != nil {
		return err
	}

	logrusctx.Info(ctx, id.Long())
	return tsk.Create(ctx)
}

func delete(ctx context.Context, cmd Command) error {
	id := common.NewRandomIdentifier(cmd.Name)
	if identifier, err := common.ParseIdentifier(cmd.Name); err == nil {
		id = identifier
	}

	tsk, err := task.New(ctx, cmd.Cloud, id, cmd.Task)
	if err != nil {
		return err
	}

	return tsk.Delete(ctx)
}

func read(ctx context.Context, cmd Command) error {
	id := common.NewRandomIdentifier(cmd.Name)
	if identifier, err := common.ParseIdentifier(cmd.Name); err == nil {
		id = identifier
	}

	tsk, err := task.New(ctx, cmd.Cloud, id, cmd.Task)
	if err != nil {
		return err
	}

	if err := tsk.Read(ctx); err != nil {
		return err
	}

	logs, err := tsk.Logs(ctx)
	if err != nil {
		return err
	}

	for _, log := range logs {
		for _, line := range strings.Split(strings.Trim(log, "\n"), "\n") {
			logrusctx.Info(ctx, line)
		}
	}

	return nil
}
