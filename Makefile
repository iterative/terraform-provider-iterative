HOSTNAME=github.com
NAMESPACE=iterative
NAME=iterative
VERSION=0.6
#OS_ARCH=linux_amd64
OS_ARCH=darwin_amd64
BINARY=terraform-provider-${NAME}

default: install

build:
	go build -o ${BINARY}

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

test: 
	go test ./... $(TESTARGS) -timeout=30s -parallel=4

testacc: 
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m
