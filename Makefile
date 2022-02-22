HOSTNAME=github.com
NAMESPACE=iterative
NAME=iterative
VERSION=0.0.${shell date +%s}+development
OS_ARCH=${shell go env GOOS}_${shell go env GOARCH}
BINARY=terraform-provider-${NAME}
INSTALL_PATH=~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

default: build

build:
	go build

install:
	GOBIN=${INSTALL_PATH} go install

test:
	go test ./... ${TESTARGS} -timeout=30s -parallel=4 -short

smoke:
	go test ./task -v ${TESTARGS} -timeout=30m -count=1

sweep:
	SMOKE_TEST_SWEEP=true go test ./task -v ${TESTARGS} -timeout=30m -count=1

testacc:
	TF_ACC=1 go test ./... -v ${TESTARGS} -timeout 120m
