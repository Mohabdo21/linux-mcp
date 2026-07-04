SHELL := /bin/bash

BINARY = linux-mcp
BUILD_DIR = bin
VERSION ?= dev
LD_FLAGS = -X main.Version=$(VERSION)

.PHONY: build build-static test check release

build:
	CGO_ENABLED=0 go build -ldflags="$(LD_FLAGS)" -o $(BUILD_DIR)/$(BINARY) .

build-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-w -s $(LD_FLAGS)" \
		-trimpath \
		-mod=readonly \
		-o $(BUILD_DIR)/$(BINARY)_static .

test: check
	@echo "Running tests..."
	go test -race -v ./...

check:
	go fmt ./...
	go fix ./...
	go vet ./...
	golangci-lint fmt
	golangci-lint run --fix

release:
	@scripts/release.sh "$(VERSION)"
