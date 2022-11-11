HOSTNAME=github.com
NAMESPACE=iterative
NAME=iterative
VERSION=0.0.${shell date +%s}+development
GOOS=${shell go env GOOS}
GOARCH=${shell go env GOARCH}
TF_PLUGIN_INSTALL_PATH=~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${GOOS}_${GOARCH}
TPI_PATH ?= $(shell pwd)
LEO_PATH ?= $(shell pwd)/cmd/leo
GO_LINK_FLAGS ?= -s -w -X terraform-provider-iterative/iterative/utils.Version=${VERSION}

SAM_TEMPLATE := packaged.yml
SAM_STACK := leo-deployment
SAM_REGION := us-east-1

default: build

.PHONY: build
build: tpi leo

.PHONY: install
install: install-tpi

.PHONY: deps
deps: # Install development dependencies
	go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.2

.PHONY: tpi
tpi:
	CGO_ENABLED=0 \
	go build -ldflags="$(GO_LINK_FLAGS)" \
	-o $(shell pwd)/terraform-provider-iterative \
	$(TPI_PATH)

.PHONY: leo
leo:
	CGO_ENABLED=0 \
	go build -ldflags="$(GO_LINK_FLAGS)" \
	-o $(shell pwd)/leo \
	$(LEO_PATH)

.PHONY: install-tpi
install-tpi:
	GOBIN=${TF_PLUGIN_INSTALL_PATH} go install -ldflags="$(GO_LINK_FLAGS)" $(TPI_PATH)

.PHONY: test
test:
	go test ./... ${TESTARGS} -timeout=30s -parallel=4

.PHONY: smoke
smoke:
	go test ./task -v ${TESTARGS} -timeout=30m -count=1 -tags=smoke

.PHONY: sweep
sweep:
	SMOKE_TEST_SWEEP=true go test ./task -v ${TESTARGS} -timeout=30m -count=1

.PHONY: generate
generate:
	go generate ./...

.PHONY: sam-build
sam-build:
	sam build

.PHONY: sam-package
sam-package: sam-build
	sam package\
		--resolve-s3\
		--output-template-file ${SAM_TEMPLATE}\
		--region ${SAM_REGION}

.PHONY: sam-deploy
sam-deploy: sam-package
	sam deploy\
		--resolve-s3\
		--region ${SAM_REGION}\
		--template-file ${SAM_TEMPLATE}\
		--stack-name ${SAM_STACK}\
		--capabilities CAPABILITY_IAM

.PHONY: sam-get-endpoint
sam-get-endpoint:
	aws cloudformation describe-stacks\
		--stack-name ${SAM_STACK}\
		--region ${SAM_REGION}\
		--query "Stacks[0].Outputs[?OutputKey=='WebSocketEndpoint'].OutputValue"\
		--output text
