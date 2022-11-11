package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

const (
	forkHeaderName  = "x-lambda-fork"
	forkHeaderValue = "true"
)

type handlerType func(context.Context, *events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error)

// forking overcomes AWS API Gateway time limits by invoking again the Lambda
// function with the same parameters asynchronously and returning immediately;
// similar to forking daemons.
func forking(ctx context.Context, handler handlerType) handlerType {
	return func(ctx context.Context, req *events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
		if req.Headers == nil {
			req.Headers = make(map[string]string)
		}

		if val, ok := req.Headers[forkHeaderName]; ok && val == forkHeaderValue {
			return handler(ctx, req)
		}

		req.Headers[forkHeaderName] = forkHeaderValue

		payload, err := json.Marshal(req)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}

		input := &lambda.InvokeInput{
			FunctionName:   aws.String(lambdacontext.FunctionName),
			InvocationType: types.InvocationTypeEvent,
			Payload:        payload,
		}

		if _, err = lambdaClient.Invoke(ctx, input); err != nil {
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}

		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
	}
}
