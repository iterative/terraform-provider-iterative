package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
)

type WebSocketWriter struct {
	ConnnectionID string
}

func (w WebSocketWriter) Write(data []byte) (int, error) {
	_, err := apiGatewayManagementAPIClient.PostToConnection(context.Background(), &apigatewaymanagementapi.PostToConnectionInput{
		Data:         data,
		ConnectionId: aws.String(w.ConnnectionID),
	})

	return len(data), err
}
