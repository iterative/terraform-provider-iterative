package main

import (
	"context"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
)

var cfg aws.Config
var lambdaClient *lambda.Client
var apiGatewayManagementAPIClient *apigatewaymanagementapi.Client

func init() {
	var err error
	cfg, err = config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}

	lambdaClient = lambda.NewFromConfig(cfg)
}

func initAPIGatewayManagementAPIClient(req *events.APIGatewayWebsocketProxyRequest) {
	if apiGatewayManagementAPIClient == nil {
		cp := cfg.Copy()

		// https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/endpoints
		cp.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			var endpoint url.URL
			endpoint.Path = req.RequestContext.Stage
			endpoint.Host = req.RequestContext.DomainName
			endpoint.Scheme = "https"
			return aws.Endpoint{
				SigningRegion: region,
				URL:           endpoint.String(),
			}, nil
		})

		apiGatewayManagementAPIClient = apigatewaymanagementapi.NewFromConfig(cp)
	}
}
