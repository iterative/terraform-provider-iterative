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

default: build

.PHONY: build
build: tpi leo

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

.PHONY: install_tpi
install_tpi:
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

.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v ${TESTARGS} -timeout 120m
