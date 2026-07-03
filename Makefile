SHELL := /bin/bash

BINARY = linux-mcp
BUILD_DIR = bin

.PHONY: build build-static test check

build:
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY) .

build-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-w -s" \
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
