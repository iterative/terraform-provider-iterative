HOSTNAME=github.com
NAMESPACE=iterative
NAME=iterative
VERSION=0.0.0+development
OS_ARCH=${shell go env GOOS}_${shell go env GOARCH}
BINARY=terraform-provider-${NAME}
INSTALL_PATH=~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

default: install

build:
	go build

install:
	GOBIN=${INSTALL_PATH} go install

test:
	go test ./... ${TESTARGS} -timeout=30s -parallel=4

testacc:
	TF_ACC=1 go test ./... -v ${TESTARGS} -timeout 120m
