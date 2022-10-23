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

LEO_BUILD_COMMAND ?= \
	CGO_ENABLED=0 \
	go build -ldflags="$(GO_LINK_FLAGS)" \
	-o $(shell pwd)/leo-$(GOOS)-$(GOARCH) \
	$(LEO_PATH)

default: build

# TODO: Should this be renamed to something like build-check ?
.PHONY: build
build:
	go build ./...

.PHONY: leo-bin
leo-bin:
	$(LEO_BUILD_COMMAND)

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

.PHONY: targets
targets:
	@awk -F: '/^[^ \t="]+:/ && !/PHONY/ {print $$1}' Makefile | sort -u
