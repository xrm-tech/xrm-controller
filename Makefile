all: test build

VERSION ?= $(shell git describe --abbrev=4 --dirty --always --tags)

DOCKER ?= docker
GO ?= go

SRCS:=$(shell find . -name '*.go' | grep -v 'vendor')

## help: Prints a list of available build targets.
help:
	echo "Usage: make <OPTIONS> ... <TARGETS>"
	echo ""
	echo "Available targets are:"
	echo ''
	sed -n 's/^##//p' ${PWD}/Makefile | column -t -s ':' | sed -e 's/^/ /'
	echo
	echo "Targets run by default are: `sed -n 's/^all: //p' ./Makefile | sed -e 's/ /, /g' | sed -e 's/\(.*\), /\1, and /'`"

## clean: Removes any previously created build artifacts.
clean:
	rm -f ./xrm-controller

build: FORCE
	CGO_ENABLED=0 GO111MODULE=on ${GO} build -ldflags '-X main.BuildVersion=$(VERSION)' ${PWD}/cmd/xrm-controller
	CGO_ENABLED=0 GO111MODULE=on ${GO} build -ldflags '-X main.BuildVersion=$(VERSION)' ${PWD}/cmd/xrm-cli

debug: FORCE
	GO111MODULE=on ${GO} build -gcflags=all='-N -l' -ldflags '-X main.BuildVersion=$(VERSION)' ${PWD}/cmd/xrm-controller
	GO111MODULE=on ${GO} build -gcflags=all='-N -l' -ldflags '-X main.BuildVersion=$(VERSION)' ${PWD}/cmd/xrm-cli

## format: Applies Go formatting to code.
format:
	${GO} fmt ./...

## test: Executes any unit tests.
test:
	${GO} test -cover -race ./...

integrations:
	${GO} test -count=1 -tags=test_integration ./...

lint:
	golangci-lint run

FORCE:

.PHONY: build
